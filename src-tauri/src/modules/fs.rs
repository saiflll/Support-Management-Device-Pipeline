use serde::{Deserialize, Serialize};
use std::fs;
use std::path::Path;

#[derive(Clone, Serialize, Deserialize, Debug)]
pub struct FileEntry {
    pub name: String,
    pub path: String,
    pub is_dir: bool,
    pub size: u64,
    pub modified: String,
}

#[tauri::command]
pub fn fs_list_dir(path: String) -> Result<Vec<FileEntry>, String> {
    let dir = Path::new(&path);
    if !dir.is_dir() {
        return Err(format!("Not a directory: {}", path));
    }

    let mut entries = Vec::new();
    for entry in fs::read_dir(dir).map_err(|e| e.to_string())? {
        let entry = entry.map_err(|e| e.to_string())?;
        let metadata = entry.metadata().map_err(|e| e.to_string())?;
        let modified = metadata
            .modified()
            .ok()
            .map(|t| {
                let dt: chrono::DateTime<chrono::Local> = t.into();
                dt.format("%Y-%m-%d %H:%M:%S").to_string()
            })
            .unwrap_or_default();

        entries.push(FileEntry {
            name: entry.file_name().to_string_lossy().to_string(),
            path: entry.path().to_string_lossy().to_string(),
            is_dir: metadata.is_dir(),
            size: metadata.len(),
            modified,
        });
    }

    entries.sort_by(|a, b| {
        if a.is_dir != b.is_dir {
            b.is_dir.cmp(&a.is_dir)
        } else {
            a.name.to_lowercase().cmp(&b.name.to_lowercase())
        }
    });

    Ok(entries)
}

#[tauri::command]
pub fn fs_read_file(path: String) -> Result<String, String> {
    fs::read_to_string(&path).map_err(|e| e.to_string())
}

#[tauri::command]
pub fn fs_write_file(path: String, content: String) -> Result<(), String> {
    if let Some(parent) = Path::new(&path).parent() {
        fs::create_dir_all(parent).map_err(|e| e.to_string())?;
    }
    fs::write(&path, &content).map_err(|e| e.to_string())
}

#[tauri::command]
pub fn fs_create_dir(path: String) -> Result<(), String> {
    fs::create_dir_all(&path).map_err(|e| e.to_string())
}

#[tauri::command]
pub fn fs_delete(path: String) -> Result<(), String> {
    let p = Path::new(&path);
    if p.is_dir() {
        fs::remove_dir_all(p).map_err(|e| e.to_string())
    } else {
        fs::remove_file(p).map_err(|e| e.to_string())
    }
}

#[tauri::command]
pub fn fs_rename(from: String, to: String) -> Result<(), String> {
    fs::rename(&from, &to).map_err(|e| e.to_string())
}

#[tauri::command]
pub fn fs_exists(path: String) -> bool {
    Path::new(&path).exists()
}

#[tauri::command]
pub fn fs_stat(path: String) -> Result<FileEntry, String> {
    let p = Path::new(&path);
    let metadata = fs::metadata(p).map_err(|e| e.to_string())?;
    let modified = metadata
        .modified()
        .ok()
        .map(|t| {
            let dt: chrono::DateTime<chrono::Local> = t.into();
            dt.format("%Y-%m-%d %H:%M:%S").to_string()
        })
        .unwrap_or_default();

    Ok(FileEntry {
        name: p
            .file_name()
            .map(|n| n.to_string_lossy().to_string())
            .unwrap_or_default(),
        path: p.to_string_lossy().to_string(),
        is_dir: metadata.is_dir(),
        size: metadata.len(),
        modified,
    })
}

#[tauri::command]
pub fn sys_check_installed(cmd: String) -> bool {
    let mut check_cmd = if cfg!(target_os = "windows") {
        let mut c = std::process::Command::new("powershell.exe");
        c.args(&["-Command", &format!("Get-Command {} -ErrorAction SilentlyContinue", cmd)]);
        c
    } else {
        let mut c = std::process::Command::new("which");
        c.arg(&cmd);
        c
    };

    #[cfg(target_os = "windows")]
    {
        use std::os::windows::process::CommandExt;
        check_cmd.creation_flags(0x08000000); // CREATE_NO_WINDOW
    }

    check_cmd.status().map(|s| s.success()).unwrap_or(false)
}

#[derive(Clone, Serialize, Deserialize, Debug)]
pub struct GitStatusResult {
    pub branch: String,
    pub untracked: Vec<String>,
    pub modified: Vec<String>,
    pub staged: Vec<String>,
}

#[tauri::command]
pub fn git_get_status(repo_path: String) -> Result<GitStatusResult, String> {
    let mut branch_cmd = std::process::Command::new("git");
    branch_cmd.args(&["rev-parse", "--abbrev-ref", "HEAD"]).current_dir(&repo_path);
    
    let mut status_cmd = std::process::Command::new("git");
    status_cmd.args(&["status", "--porcelain"]).current_dir(&repo_path);

    #[cfg(target_os = "windows")]
    {
        use std::os::windows::process::CommandExt;
        branch_cmd.creation_flags(0x08000000);
        status_cmd.creation_flags(0x08000000);
    }

    let branch_output = branch_cmd.output().map_err(|e| e.to_string())?;
    let branch = String::from_utf8_lossy(&branch_output.stdout).trim().to_string();

    let status_output = status_cmd.output().map_err(|e| e.to_string())?;
    let status_str = String::from_utf8_lossy(&status_output.stdout);
    
    let mut untracked = Vec::new();
    let mut modified = Vec::new();
    let mut staged = Vec::new();

    for line in status_str.lines() {
        if line.len() < 3 { continue; }
        let code = &line[0..2];
        let file = line[3..].trim().to_string();
        
        match code {
            "??" => untracked.push(file),
            " M" | " D" => modified.push(file),
            "M " | "A " | "D " => staged.push(file),
            _ => modified.push(file),
        }
    }

    Ok(GitStatusResult {
        branch: if branch.is_empty() { "detached".to_string() } else { branch },
        untracked,
        modified,
        staged,
    })
}

#[tauri::command]
pub fn git_commit(repo_path: String, message: String) -> Result<String, String> {
    let mut add_cmd = std::process::Command::new("git");
    add_cmd.args(&["add", "."]).current_dir(&repo_path);

    let mut commit_cmd = std::process::Command::new("git");
    commit_cmd.args(&["commit", "-m", &message]).current_dir(&repo_path);

    #[cfg(target_os = "windows")]
    {
        use std::os::windows::process::CommandExt;
        add_cmd.creation_flags(0x08000000);
        commit_cmd.creation_flags(0x08000000);
    }

    let add_status = add_cmd.status().map_err(|e| e.to_string())?;
    if !add_status.success() {
        return Err("Failed to stage changes with git add".to_string());
    }

    let commit_output = commit_cmd.output().map_err(|e| e.to_string())?;
    if commit_output.status.success() {
        Ok(String::from_utf8_lossy(&commit_output.stdout).trim().to_string())
    } else {
        Err(String::from_utf8_lossy(&commit_output.stderr).trim().to_string())
    }
}

#[tauri::command]
pub fn git_stage_file(repo_path: String, file_path: String) -> Result<(), String> {
    let mut cmd = std::process::Command::new("git");
    cmd.args(&["add", &file_path]).current_dir(&repo_path);

    #[cfg(target_os = "windows")]
    {
        use std::os::windows::process::CommandExt;
        cmd.creation_flags(0x08000000);
    }

    let status = cmd.status().map_err(|e| e.to_string())?;
    if status.success() {
        Ok(())
    } else {
        Err(format!("Failed to stage file: {}", file_path))
    }
}

#[tauri::command]
pub fn git_unstage_file(repo_path: String, file_path: String) -> Result<(), String> {
    let mut cmd = std::process::Command::new("git");
    cmd.args(&["restore", "--staged", &file_path]).current_dir(&repo_path);

    #[cfg(target_os = "windows")]
    {
        use std::os::windows::process::CommandExt;
        cmd.creation_flags(0x08000000);
    }

    let status = cmd.status().map_err(|e| e.to_string())?;
    if status.success() {
        Ok(())
    } else {
        Err(format!("Failed to unstage file: {}", file_path))
    }
}

#[tauri::command]
pub fn git_discard_file(repo_path: String, file_path: String) -> Result<(), String> {
    let path = std::path::Path::new(&repo_path).join(&file_path);
    
    let mut restore_cmd = std::process::Command::new("git");
    restore_cmd.args(&["restore", &file_path]).current_dir(&repo_path);

    #[cfg(target_os = "windows")]
    {
        use std::os::windows::process::CommandExt;
        restore_cmd.creation_flags(0x08000000);
    }

    let restore_status = restore_cmd.status().map_err(|e| e.to_string())?;
    
    if !restore_status.success() {
        if path.exists() {
            if path.is_dir() {
                std::fs::remove_dir_all(&path).map_err(|e| e.to_string())?;
            } else {
                std::fs::remove_file(&path).map_err(|e| e.to_string())?;
            }
        }
    }
    Ok(())
}

#[tauri::command]
pub fn git_stage_all(repo_path: String) -> Result<(), String> {
    let mut cmd = std::process::Command::new("git");
    cmd.args(&["add", "."]).current_dir(&repo_path);

    #[cfg(target_os = "windows")]
    {
        use std::os::windows::process::CommandExt;
        cmd.creation_flags(0x08000000);
    }

    let status = cmd.status().map_err(|e| e.to_string())?;
    if status.success() {
        Ok(())
    } else {
        Err("Failed to stage all files".to_string())
    }
}

#[tauri::command]
pub fn git_unstage_all(repo_path: String) -> Result<(), String> {
    let mut cmd = std::process::Command::new("git");
    cmd.args(&["reset"]).current_dir(&repo_path);

    #[cfg(target_os = "windows")]
    {
        use std::os::windows::process::CommandExt;
        cmd.creation_flags(0x08000000);
    }

    let status = cmd.status().map_err(|e| e.to_string())?;
    if status.success() {
        Ok(())
    } else {
        Err("Failed to unstage all changes".to_string())
    }
}
