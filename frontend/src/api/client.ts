export class ApiError extends Error {
  readonly status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

let baseUrl = import.meta.env.VITE_ENGINE_URL ?? 'http://127.0.0.1:8787';
let token = import.meta.env.VITE_ENGINE_TOKEN ?? '';

/** Called by the desktop shell (Tauri sidecar handshake) to point the UI at the engine. */
export function configureEngine(cfg: { url: string; token: string }): void {
  baseUrl = cfg.url;
  token = cfg.token;
}
export const engineUrl = (): string => baseUrl;

export async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`${baseUrl}${path}`, {
      method,
      headers: { Authorization: `Bearer ${token}`, ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}) },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });
  } catch {
    throw new ApiError(0, 'Engine unreachable. Is xolographd running?');
  }
  const data: unknown = await res.json().catch(() => null);
  if (!res.ok) {
    const msg = (data as { error?: string } | null)?.error ?? res.statusText;
    throw new ApiError(res.status, msg);
  }
  return data as T;
}
