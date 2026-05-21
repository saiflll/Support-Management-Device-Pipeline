mod modules;

use modules::{ai, fs, pio, pty};

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    env_logger::init();

    tauri::Builder::default()
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_fs::init())
        .manage(pty::PtyManager::new())
        .manage(ai::AiManager::new())
        .invoke_handler(tauri::generate_handler![
            pty::pty_open,
            pty::pty_write,
            pty::pty_resize,
            pty::pty_close,
            pty::pty_list,
            ai::ai_chat,
            ai::ai_auto_route,
            ai::ai_list_agents,
            ai::ai_update_agent,
            ai::ai_remove_agent,
            ai::ai_validate_agent,
            ai::ai_set_provider,
            fs::fs_list_dir,
            fs::fs_read_file,
            fs::fs_write_file,
            fs::fs_create_dir,
            fs::fs_delete,
            fs::fs_rename,
            fs::fs_exists,
            fs::fs_stat,
            fs::sys_check_installed,
            fs::git_get_status,
            fs::git_commit,
            fs::git_stage_file,
            fs::git_unstage_file,
            fs::git_discard_file,
            fs::git_stage_all,
            fs::git_unstage_all,
            pio::pio_detect,
            pio::pio_install,
            pio::pio_init,
            pio::pio_run,
            pio::pio_list_boards,
            ai::ai_get_usage,
            ai::ai_reset_usage,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
