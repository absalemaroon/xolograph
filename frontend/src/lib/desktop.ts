import { configureEngine } from '@/api/client';
import { downloadText } from './download';

/** True inside the Tauri desktop shell; false in a plain browser (dev/preview). */
export const isDesktop = (): boolean => typeof window !== 'undefined' && '__TAURI_INTERNALS__' in window;

async function call<T>(cmd: string, args?: Record<string, unknown>): Promise<T> {
  const core = await import('@tauri-apps/api/core');
  return core.invoke<T>(cmd, args);
}

/** Desktop: wait for the sidecar engine's handshake and point the API client at it. */
export async function bootEngine(timeoutMs = 20000): Promise<void> {
  if (!isDesktop()) return;
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const cfg = await call<{ url: string; token: string } | null>('engine_config'); // rejects with a message if the engine failed
    if (cfg) { configureEngine(cfg); return; }
    await new Promise((r) => setTimeout(r, 200));
  }
  throw new Error('The engine did not start in time.');
}

export async function pickEvidenceFile(): Promise<string | null> {
  return isDesktop() ? call<string | null>('pick_evidence_file') : null;
}

export async function dataLocation(): Promise<string | null> {
  return isDesktop() ? call<string>('data_location') : null;
}

/** Desktop: native Save dialog. Browser: standard download. Returns where it was saved, if known. */
export async function saveTextFile(name: string, contents: string, mime: string): Promise<{ saved: boolean; path?: string }> {
  if (!isDesktop()) {
    downloadText(name, contents, mime);
    return { saved: true };
  }
  const path = await call<string | null>('save_text_file', { defaultName: name, contents });
  return path ? { saved: true, path } : { saved: false };
}

export interface UpdateInfo { version: string; notes?: string | null; date?: string | null }

export const pickSavePath = (defaultName: string): Promise<string | null> =>
  isDesktop() ? call<string | null>('pick_save_path', { defaultName }) : Promise.resolve(null);

export const pickBackupFile = (): Promise<string | null> =>
  isDesktop() ? call<string | null>('pick_backup_file') : Promise.resolve(null);

/** Stops the engine and relaunches the app (applies a staged restore). */
export async function restartApp(): Promise<void> {
  await call<void>('restart_app');
}

export const checkUpdate = (): Promise<UpdateInfo | null> => call<UpdateInfo | null>('check_update');
export const installUpdate = (): Promise<void> => call<void>('install_update');
