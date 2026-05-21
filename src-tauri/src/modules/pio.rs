use serde::{Deserialize, Serialize};
use std::process::Command;

#[derive(Serialize, Deserialize, Debug)]
pub struct PioStatus {
    pub installed: bool,
    pub version: Option<String>,
    pub python: Option<String>,
    pub pio_path: Option<String>,
    pub error: Option<String>,
}

#[derive(Serialize, Deserialize, Debug)]
pub struct PioResult {
    pub success: bool,
    pub output: String,
    pub error: Option<String>,
}

/// Try to find a working way to execute PlatformIO commands.
/// Returns (executable, base_args) or None if not found.
/// Priority: compatible Python via `-m platformio` > binary on PATH > generic Python.
fn find_pio() -> Option<(String, Vec<String>)> {
    // 1. Try `py -3.12/3.13 -m platformio` first — preferred compatible Python versions
    let py_candidates = [
        ("py", &["-3.12", "-m", "platformio"][..]),
        ("py", &["-3.13", "-m", "platformio"][..]),
        ("python3.12", &["-m", "platformio"]),
        ("python3.13", &["-m", "platformio"]),
    ];

    for (exe, args) in &py_candidates {
        if let Ok(output) = Command::new(exe).args(*args).arg("--version").output() {
            if output.status.success() {
                return Some((exe.to_string(), args.iter().map(|s| s.to_string()).collect()));
            }
        }
    }

    // 2. Try direct `pio` / `platformio` binary via PATH
    for bin in &["pio", "platformio"] {
        if let Ok(output) = Command::new(bin).arg("--version").output() {
            if output.status.success() {
                return Some((bin.to_string(), vec![]));
            }
        }
    }

    // 3. Fallback: generic Python module
    let fallback_candidates = [
        ("python", &["-m", "platformio"][..]),
        ("python3", &["-m", "platformio"]),
        ("py", &["-m", "platformio"]),
    ];

    for (exe, args) in &fallback_candidates {
        if let Ok(output) = Command::new(exe).args(*args).arg("--version").output() {
            if output.status.success() {
                return Some((exe.to_string(), args.iter().map(|s| s.to_string()).collect()));
            }
        }
    }

    None
}

fn run_pio(args: &[&str]) -> Result<std::process::Output, String> {
    let (exe, base_args) = find_pio().ok_or_else(|| {
        "PlatformIO not found. Install it with: pip install platformio".to_string()
    })?;

    let mut cmd = Command::new(&exe);
    cmd.args(&base_args);
    cmd.args(args);
    cmd.output().map_err(|e| format!("Failed to run pio: {}", e))
}

/// Check if a Python version string (e.g. "Python 3.14.3") is compatible with PlatformIO.
fn python_version_compatible(ver: &str) -> bool {
    // Extract major.minor, e.g. "3.14" from "Python 3.14.3"
    let num_part = ver.trim_start_matches("Python ").trim();
    let parts: Vec<&str> = num_part.split('.').collect();
    if parts.len() < 2 { return false; }
    let major: u32 = parts[0].parse().unwrap_or(0);
    let minor: u32 = parts[1].parse().unwrap_or(0);
    major == 3 && minor >= 10 && minor <= 13
}

/// Try to find a compatible Python (3.10–3.13) and return (exe, version_string).
fn find_compatible_python() -> Option<(String, String)> {
    let py_candidates = [
        "py -3.12", "py -3.13", "py -3.10", "py -3.11",
        "python3.12", "python3.13", "python3.10", "python3.11",
        "python", "python3", "py",
    ];
    for entry in &py_candidates {
        let parts: Vec<&str> = entry.split_whitespace().collect();
        let exe = parts[0];
        let extra: Vec<&str> = parts[1..].to_vec();

        let mut cmd = Command::new(exe);
        for a in &extra { cmd.arg(a); }
        cmd.arg("--version");

        if let Ok(output) = cmd.output() {
            if output.status.success() {
                let ver = String::from_utf8_lossy(&output.stdout).trim().to_string();
                if !ver.is_empty() && python_version_compatible(&ver) {
                    return Some((entry.to_string(), ver));
                }
                // Also check stderr (py launcher sometimes outputs version to stderr)
                let ver2 = String::from_utf8_lossy(&output.stderr).trim().to_string();
                if !ver2.is_empty() && python_version_compatible(&ver2) {
                    return Some((entry.to_string(), ver2));
                }
            }
        }
    }
    None
}

#[tauri::command]
pub fn pio_detect() -> Result<PioStatus, String> {
    let mut status = PioStatus {
        installed: false,
        version: None,
        python: None,
        pio_path: None,
        error: None,
    };

    // Prefer a compatible Python version (3.10–3.13) over the system default
    // This avoids flagging Python 3.14+ as incompatible when 3.12/3.13 is also installed.
    let current_py = if let Some((ref _entry, ref ver)) = find_compatible_python() {
        Some(ver.clone())
    } else {
        let mut found = None;
        for py in &["python", "python3", "py"] {
            if let Ok(output) = Command::new(py).arg("--version").output() {
                if output.status.success() {
                    let v = String::from_utf8_lossy(&output.stdout).trim().to_string();
                    if !v.is_empty() { found = Some(v); break; }
                    let v2 = String::from_utf8_lossy(&output.stderr).trim().to_string();
                    if !v2.is_empty() { found = Some(v2); break; }
                }
            }
        }
        found
    };

    if let Some(ref py_ver) = current_py {
        status.python = Some(py_ver.clone());
        if !python_version_compatible(py_ver) {
            status.error = Some(format!(
                "Python {} is not supported by PlatformIO. Need Python 3.10–3.13.\nInstall Python 3.12: https://www.python.org/downloads/",
                py_ver
            ));
        }
    }

    match find_pio() {
        Some((ref exe, ref base_args)) => {
            let mut cmd = Command::new(exe);
            cmd.args(base_args);
            cmd.arg("--version");

            if let Ok(output) = cmd.output() {
                if output.status.success() {
                    status.installed = true;
                    status.pio_path = Some(
                        if base_args.is_empty() { exe.clone() }
                        else { format!("{} {}", exe, base_args.join(" ")) }
                    );
                    status.version = Some(
                        String::from_utf8_lossy(&output.stdout).trim().to_string(),
                    );
                } else {
                    // pio ran but exited with error — capture stderr for diagnosis
                    let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
                    if !stderr.is_empty() {
                        status.error = Some(stderr);
                    } else if status.error.is_none() {
                        status.error = Some("PlatformIO exited with an unknown error. Python version may be incompatible.".to_string());
                    }
                }
            }
        }
        None => {
            // PIO not found at all
            if status.error.is_none() {
                if let Some(ref py_ver) = current_py {
                    if python_version_compatible(py_ver) {
                        status.error = Some("PlatformIO not found. Click Install to install it.".to_string());
                    }
                } else {
                    status.error = Some("Python not found. Install Python 3.12 first.".to_string());
                }
            }
        }
    }

    Ok(status)
}

#[tauri::command]
pub fn pio_install() -> Result<PioResult, String> {
    // Find a compatible Python to install with
    let (install_exe, install_args) = if let Some((ref py_entry, ref _ver)) = find_compatible_python() {
        let parts: Vec<&str> = py_entry.split_whitespace().collect();
        let exe = parts[0];
        let extra = &parts[1..];
        let mut args: Vec<String> = extra.iter().map(|s| s.to_string()).collect();
        args.push("-m".to_string());
        args.push("pip".to_string());
        args.push("install".to_string());
        args.push("platformio".to_string());
        (exe.to_string(), args)
    } else if cfg!(target_os = "windows") {
        ("pip".to_string(), vec!["install".to_string(), "platformio".to_string()])
    } else {
        ("pip3".to_string(), vec!["install".to_string(), "platformio".to_string()])
    };

    let mut cmd = Command::new(&install_exe);
    cmd.args(&install_args);
    let result = cmd.output().map_err(|e| format!("Failed to run pip: {}", e))?;

    Ok(PioResult {
        success: result.status.success(),
        output: String::from_utf8_lossy(&result.stdout).to_string(),
        error: if result.status.success() {
            None
        } else {
            Some(String::from_utf8_lossy(&result.stderr).to_string())
        },
    })
}

#[tauri::command]
pub fn pio_init(path: String, board: Option<String>) -> Result<PioResult, String> {
    let mut args = vec!["project", "init", "-d", &path];

    if let Some(ref b) = board {
        args.extend_from_slice(&["-b", b]);
    }

    let output = run_pio(&args)?;

    Ok(PioResult {
        success: output.status.success(),
        output: String::from_utf8_lossy(&output.stdout).to_string(),
        error: if output.status.success() {
            None
        } else {
            Some(String::from_utf8_lossy(&output.stderr).to_string())
        },
    })
}

#[tauri::command]
pub fn pio_run(target: String, directory: String) -> Result<PioResult, String> {
    let args = ["run", "-d", &directory, "-t", &target];
    let output = run_pio(&args)?;

    Ok(PioResult {
        success: output.status.success(),
        output: String::from_utf8_lossy(&output.stdout).to_string(),
        error: if output.status.success() {
            None
        } else {
            Some(String::from_utf8_lossy(&output.stderr).to_string())
        },
    })
}

#[tauri::command]
pub fn pio_list_boards(search: Option<String>) -> Result<Vec<String>, String> {
    let args = ["boards"];
    let output = run_pio(&args)?;

    if !output.status.success() {
        return Err(String::from_utf8_lossy(&output.stderr).to_string());
    }

    let out = String::from_utf8_lossy(&output.stdout);
    let mut boards: Vec<String> = Vec::new();

    for line in out.lines().skip(2) {
        let line = line.trim();
        if line.is_empty() {
            continue;
        }
        // Extract only the board ID (first whitespace-separated token)
        let board_id = line.split_whitespace().next().unwrap_or(line);
        if let Some(ref search_term) = search {
            if board_id.to_lowercase().contains(&search_term.to_lowercase()) {
                boards.push(board_id.to_string());
            }
        } else {
            boards.push(board_id.to_string());
        }
    }

    Ok(boards)
}
