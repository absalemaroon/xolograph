import { useState } from 'react';
import pkg from '../../../package.json';
import { Badge, Button, Panel } from '@/components/ui';
import { checkUpdate, installUpdate, isDesktop } from '@/lib/desktop';
import type { UpdateInfo } from '@/lib/desktop';

type UpdateState =
  | { phase: 'idle' }
  | { phase: 'checking' }
  | { phase: 'current' }
  | { phase: 'available'; info: UpdateInfo }
  | { phase: 'installing' }
  | { phase: 'error'; message: string };

export function UpdatePanel() {
  const [state, setState] = useState<UpdateState>({ phase: 'idle' });
  if (!isDesktop()) return null;

  const check = async (): Promise<void> => {
    setState({ phase: 'checking' });
    try {
      const info = await checkUpdate();
      setState(info ? { phase: 'available', info } : { phase: 'current' });
    } catch (e) {
      setState({ phase: 'error', message: e instanceof Error ? e.message : String(e) });
    }
  };

  const install = async (): Promise<void> => {
    setState({ phase: 'installing' });
    try {
      await installUpdate(); // restarts the app on success
    } catch (e) {
      setState({ phase: 'error', message: e instanceof Error ? e.message : String(e) });
    }
  };

  return (
    <Panel title="Updates" actions={<Button size="sm" loading={state.phase === 'checking'} disabled={state.phase === 'installing'} onClick={() => void check()}>Check for updates</Button>}>
      <div className="space-y-3 text-xs">
        <div className="flex justify-between"><span className="text-fg-muted">Installed version</span><span>v{pkg.version}</span></div>
        {state.phase === 'current' && <p>● You are up to date.</p>}
        {state.phase === 'available' && (
          <div className="space-y-2">
            <p><Badge tone="outline">New</Badge> Version {state.info.version} is available.</p>
            {state.info.notes && <p className="whitespace-pre-wrap text-fg-soft">{state.info.notes}</p>}
            <Button variant="primary" onClick={() => void install()}>Install &amp; restart</Button>
          </div>
        )}
        {state.phase === 'installing' && <p className="animate-pulse">Downloading and verifying the signed update…</p>}
        {state.phase === 'error' && <p>▲ {state.message}</p>}
        <p className="text-[11px] text-fg-muted">Updates are checked only when you press the button, and are accepted only if signed with the project's key.</p>
      </div>
    </Panel>
  );
}
