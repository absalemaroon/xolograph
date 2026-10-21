//! Xolograph desktop shell.
//!
//! Responsibilities:
//!  * start the Go engine (`xolographd`) as a sidecar, with the app data directory,
//!  * read its one-line JSON handshake (loopback address + random per-run token),
//!  * expose that to the UI, plus native open/save dialogs,
//!  * stop the engine when the app exits.

use std::sync::Mutex;

use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Manager, RunEvent, State};
use tauri_plugin_dialog::DialogExt;
use tauri_plugin_shell::process::{CommandChild, CommandEvent};
use tauri_plugin_shell::ShellExt;

#[derive(Clone, Serialize)]
struct EngineConfig {
    url: String,
    token: String,
}

#[derive(Deserialize)]
struct Handshake {
    addr: String,
    token: String,
}

#[derive(Default)]
struct EngineState {
    config: Mutex<Option<EngineConfig>>,
    error: Mutex<Option<String>>,
    child: Mutex<Option<CommandChild>>,
}

fn stop_engine(app: &AppHandle) {
    let child = app.state::<EngineState>().child.lock().unwrap().take();
    if let Some(child) = child {
        let _ = child.kill();
    }
}

fn set_error(app: &AppHandle, msg: String) {
    let state = app.state::<EngineState>();
    *state.config.lock().unwrap() = None;
    *state.error.lock().unwrap() = Some(msg);
}

fn start_engine(app: &AppHandle) -> Result<(), String> {
    let data_dir = app.path().app_data_dir().map_err(|e| e.to_string())?;
    std::fs::create_dir_all(&data_dir).map_err(|e| e.to_string())?;
    let dir = data_dir.to_string_lossy().to_string();

    let (mut rx, child) = app
        .shell()
        .sidecar("xolographd")
        .map_err(|e| e.to_string())?
        .args(["--data-dir", dir.as_str(), "--exit-on-stdin-close"])
        .spawn()
        .map_err(|e| e.to_string())?;
    *app.state::<EngineState>().child.lock().unwrap() = Some(child);

    let handle = app.clone();
    tauri::async_runtime::spawn(async move {
        let mut ready = false;
        while let Some(event) = rx.recv().await {
            match event {
                CommandEvent::Stdout(line) if !ready => {
                    let text = String::from_utf8_lossy(&line).to_string();
                    if let Ok(h) = serde_json::from_str::<Handshake>(text.trim()) {
                        ready = true;
                        let state = handle.state::<EngineState>();
                        *state.config.lock().unwrap() = Some(EngineConfig {
                            url: format!("http://{}", h.addr),
                            token: h.token,
                        });
                    }
                }
                CommandEvent::Stderr(line) => {
                    eprintln!("[engine] {}", String::from_utf8_lossy(&line).trim_end());
                }
                CommandEvent::Error(e) => set_error(&handle, format!("Engine error: {e}")),
                CommandEvent::Terminated(p) => {
                    set_error(&handle, format!("The engine stopped unexpectedly (exit code {:?}).", p.code));
                }
                _ => {}
            }
        }
    });
    Ok(())
}

/// `Ok(None)` while the engine is still starting; `Err` if it failed.
#[tauri::command]
fn engine_config(state: State<'_, EngineState>) -> Result<Option<EngineConfig>, String> {
    if let Some(e) = state.error.lock().unwrap().clone() {
        return Err(e);
    }
    Ok(state.config.lock().unwrap().clone())
}

#[tauri::command]
fn data_location(app: AppHandle) -> Result<String, String> {
    app.path()
        .app_data_dir()
        .map(|p| p.to_string_lossy().to_string())
        .map_err(|e| e.to_string())
}

/// Native file picker for evidence import. `None` if the user cancels.
#[tauri::command]
async fn pick_evidence_file(app: AppHandle) -> Option<String> {
    app.dialog()
        .file()
        .set_title("Select evidence file")
        .blocking_pick_file()
        .and_then(|p| p.into_path().ok())
        .map(|p| p.to_string_lossy().to_string())
}

/// Native "save as" dialog that only returns the chosen path (used for backups, which the
/// engine writes itself).
#[tauri::command]
async fn pick_save_path(app: AppHandle, default_name: String) -> Option<String> {
    let name = default_name.replace(['/', '\\'], "_");
    app.dialog()
        .file()
        .add_filter("Xolograph backup", &["zip"])
        .set_file_name(name)
        .blocking_save_file()
        .and_then(|p| p.into_path().ok())
        .map(|p| p.to_string_lossy().to_string())
}

/// Native picker for a backup archive to restore.
#[tauri::command]
async fn pick_backup_file(app: AppHandle) -> Option<String> {
    app.dialog()
        .file()
        .set_title("Select Xolograph backup")
        .add_filter("Xolograph backup", &["zip"])
        .blocking_pick_file()
        .and_then(|p| p.into_path().ok())
        .map(|p| p.to_string_lossy().to_string())
}

/// Stop the engine, then relaunch the app (used to apply a staged restore).
#[tauri::command]
fn restart_app(app: AppHandle) {
    stop_engine(&app);
    app.restart();
}

#[derive(Serialize)]
struct UpdateInfo {
    version: String,
    notes: Option<String>,
    date: Option<String>,
}

#[cfg(feature = "updater")]
#[tauri::command]
async fn check_update(app: AppHandle) -> Result<Option<UpdateInfo>, String> {
    use tauri_plugin_updater::UpdaterExt;
    let update = app
        .updater()
        .map_err(|e| e.to_string())?
        .check()
        .await
        .map_err(|e| e.to_string())?;
    Ok(update.map(|u| UpdateInfo {
        version: u.version.clone(),
        notes: u.body.clone(),
        date: u.date.map(|d| d.to_string()),
    }))
}

#[cfg(feature = "updater")]
#[tauri::command]
async fn install_update(app: AppHandle) -> Result<(), String> {
    use tauri_plugin_updater::UpdaterExt;
    let Some(update) = app
        .updater()
        .map_err(|e| e.to_string())?
        .check()
        .await
        .map_err(|e| e.to_string())?
    else {
        return Err("No update is available.".into());
    };
    update
        .download_and_install(|_chunk, _total| {}, || {})
        .await
        .map_err(|e| e.to_string())?;
    stop_engine(&app);
    app.restart();
}

#[cfg(not(feature = "updater"))]
#[tauri::command]
async fn check_update() -> Result<Option<UpdateInfo>, String> {
    Err("Auto-update is not enabled in this build.".into())
}

#[cfg(not(feature = "updater"))]
#[tauri::command]
async fn install_update() -> Result<(), String> {
    Err("Auto-update is not enabled in this build.".into())
}

/// Native save dialog + write. The destination is chosen by the user in a native dialog
/// opened here, so the UI layer can never write to an arbitrary path.
#[tauri::command]
async fn save_text_file(app: AppHandle, default_name: String, contents: String) -> Result<Option<String>, String> {
    let name = default_name.replace(['/', '\\'], "_");
    let Some(chosen) = app.dialog().file().set_file_name(name).blocking_save_file() else {
        return Ok(None);
    };
    let path = chosen.into_path().map_err(|e| e.to_string())?;
    std::fs::write(&path, contents).map_err(|e| e.to_string())?;
    Ok(Some(path.to_string_lossy().to_string()))
}

pub fn run() {
    let builder = tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_dialog::init());
    #[cfg(feature = "updater")]
    let builder = builder.plugin(tauri_plugin_updater::Builder::new().build());

    let app = builder
        .manage(EngineState::default())
        .invoke_handler(tauri::generate_handler![
            engine_config,
            data_location,
            pick_evidence_file,
            pick_save_path,
            pick_backup_file,
            save_text_file,
            restart_app,
            check_update,
            install_update
        ])
        .setup(|app| {
            if let Err(e) = start_engine(app.handle()) {
                set_error(app.handle(), format!("Could not start the engine: {e}"));
            }
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("failed to build the Xolograph application");

    app.run(|handle, event| {
        if let RunEvent::Exit = event {
            stop_engine(handle);
        }
    });
}
