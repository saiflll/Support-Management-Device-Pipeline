use portable_pty::{native_pty_system, ChildKiller, CommandBuilder, MasterPty, PtySize};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::io::{Read, Write};
use std::sync::mpsc::{self, Receiver, Sender};
use std::sync::{Arc, Mutex};
use tauri::{AppHandle, Emitter};
use uuid::Uuid;

#[derive(Clone, Serialize, Deserialize)]
pub struct PtyOutput {
    pub session_id: String,
    pub data: String,
}

#[derive(Clone, Serialize, Deserialize)]
pub struct PtyExited {
    pub session_id: String,
    pub exit_code: i32,
}

pub struct PtySession {
    pub id: String,
    pub killer: Mutex<Box<dyn ChildKiller + Send + Sync>>,
    pub writer: Arc<Mutex<Box<dyn Write + Send>>>,
    pub master: Mutex<Box<dyn MasterPty + Send>>,
    pub tx: Sender<String>,
    pub rows: u16,
    pub cols: u16,
}

unsafe impl Send for PtySession {}
unsafe impl Sync for PtySession {}

impl Drop for PtySession {
    fn drop(&mut self) {
        if let Ok(mut k) = self.killer.lock() {
            let _ = k.kill();
        }
    }
}

pub struct PtyManager {
    pub sessions: Arc<Mutex<HashMap<String, Arc<PtySession>>>>,
}

impl PtyManager {
    pub fn new() -> Self {
        Self {
            sessions: Arc::new(Mutex::new(HashMap::new())),
        }
    }

    pub fn create_session(
        &self,
        shell: Option<&str>,
        cwd: Option<&str>,
        rows: u16,
        cols: u16,
    ) -> Result<(String, Receiver<String>), String> {
        let pty_system = native_pty_system();
        let size = PtySize {
            rows,
            cols,
            pixel_width: 0,
            pixel_height: 0,
        };
        let pair = pty_system.openpty(size).map_err(|e| e.to_string())?;

        let shell_path = shell.unwrap_or(if cfg!(target_os = "windows") {
            "powershell.exe"
        } else {
            "/bin/bash"
        });

        let mut cmd = CommandBuilder::new(shell_path);
        cmd.env("TERM", "xterm-256color");
        if shell_path == "powershell.exe" {
            cmd.args(&[
                "-NoExit",
                "-Command",
                r#"function prompt { $p = $ExecutionContext.SessionState.Path.CurrentLocation.Path; $h = [System.Environment]::GetFolderPath('UserProfile'); if ($p.StartsWith($h)) { $p = '~' + $p.Substring($h.Length) }; $g = ''; if (Get-Command git -ErrorAction SilentlyContinue) { $s = git branch --show-current 2>$null; if ($s) { $g = ' git:(' + $s.Trim() + ')' } }; $esc = [char]27; $green = "$esc[92m"; $blue = "$esc[94m"; $gray = "$esc[90m"; $cyan = "$esc[96m"; $yellow = "$esc[93m"; $magenta = "$esc[95m"; $reset = "$esc[0m"; $u = $env:USERNAME; "$green➜  $blue$u$reset $gray@$reset $cyan$u-pc$reset $yellow$p$reset$magenta$g$reset $green❯$reset " }"#,
            ]);
        }
        if let Some(dir) = cwd {
            cmd.cwd(dir);
        }

        let mut child = pair.slave.spawn_command(cmd).map_err(|e| e.to_string())?;
        drop(pair.slave);

        let killer = child.clone_killer();
        let reader = pair.master.try_clone_reader().map_err(|e| e.to_string())?;
        let writer = Arc::new(Mutex::new(
            pair.master.take_writer().map_err(|e| e.to_string())?,
        ));

        let session_id = Uuid::new_v4().to_string();
        let (tx, rx) = mpsc::channel();

        let tx_reader = tx.clone();
        let tx_waiter = tx.clone();

        std::thread::spawn(move || {
            let mut buf = [0u8; 4096];
            let mut reader = reader;
            loop {
                match reader.read(&mut buf) {
                    Ok(0) => break,
                    Ok(n) => {
                        let data = String::from_utf8_lossy(&buf[..n]).to_string();
                        if tx_reader.send(data).is_err() {
                            break;
                        }
                    }
                    Err(_) => break,
                }
            }
        });

        std::thread::spawn(move || {
            let code = child.wait().map(|s| s.exit_code()).unwrap_or(u32::MAX);
            let _ = tx_waiter.send(format!("\r\n[process exited with code {}]\r\n", code));
        });

        let session = Arc::new(PtySession {
            id: session_id.clone(),
            killer: Mutex::new(killer),
            writer,
            master: Mutex::new(pair.master),
            tx,
            rows,
            cols,
        });

        let mut sessions = self.sessions.lock().unwrap();
        sessions.insert(session_id.clone(), session);

        Ok((session_id, rx))
    }

    pub fn write_to_session(&self, session_id: &str, data: &str) -> Result<(), String> {
        let sessions = self.sessions.lock().unwrap();
        let session = sessions.get(session_id).ok_or("Session not found")?;
        let mut writer = session.writer.lock().unwrap();
        writer.write_all(data.as_bytes()).map_err(|e| e.to_string())
    }

    pub fn resize_session(&self, session_id: &str, rows: u16, cols: u16) -> Result<(), String> {
        let sessions = self.sessions.lock().unwrap();
        let session = sessions.get(session_id).ok_or("Session not found")?;
        let size = PtySize {
            rows,
            cols,
            pixel_width: 0,
            pixel_height: 0,
        };
        let master = session.master.lock().unwrap();
        master.resize(size).map_err(|e| e.to_string())
    }

    pub fn close_session(&self, session_id: &str) -> Result<(), String> {
        let mut sessions = self.sessions.lock().unwrap();
        sessions.remove(session_id);
        Ok(())
    }

    pub fn list_sessions(&self) -> Vec<String> {
        let sessions = self.sessions.lock().unwrap();
        sessions.keys().cloned().collect()
    }
}

#[tauri::command]
pub fn pty_open(
    state: tauri::State<'_, PtyManager>,
    app: AppHandle,
    shell: Option<String>,
    cwd: Option<String>,
    rows: u16,
    cols: u16,
) -> Result<String, String> {
    let (session_id, rx) =
        state.create_session(shell.as_deref(), cwd.as_deref(), rows, cols)?;

    let app_clone = app.clone();
    let sid = session_id.clone();
    std::thread::spawn(move || {
        while let Ok(data) = rx.recv() {
            let _ = app_clone.emit(
                "pty-output",
                PtyOutput {
                    session_id: sid.clone(),
                    data,
                },
            );
        }
        let _ = app_clone.emit(
            "pty-exited",
            PtyExited {
                session_id: sid.clone(),
                exit_code: 0,
            },
        );
    });

    Ok(session_id)
}

#[tauri::command]
pub fn pty_write(
    state: tauri::State<'_, PtyManager>,
    session_id: String,
    data: String,
) -> Result<(), String> {
    state.write_to_session(&session_id, &data)
}

#[tauri::command]
pub fn pty_resize(
    state: tauri::State<'_, PtyManager>,
    session_id: String,
    rows: u16,
    cols: u16,
) -> Result<(), String> {
    state.resize_session(&session_id, rows, cols)
}

#[tauri::command]
pub fn pty_close(
    state: tauri::State<'_, PtyManager>,
    session_id: String,
) -> Result<(), String> {
    state.close_session(&session_id)
}

#[tauri::command]
pub fn pty_list(state: tauri::State<'_, PtyManager>) -> Vec<String> {
    state.list_sessions()
}
