import { engineUrl } from '@/api/client';
import { Badge, Button, Input, Label, PageHeader, Panel, Segmented } from '@/components/ui';
import { useEngine } from '@/state/engineStore';
import { useSettings } from '@/state/settingsStore';
import type { Density, ThemeMode } from '@/state/settingsStore';

const SHORTCUTS: [string, string][] = [['Alt + 1…9', 'Switch view (order of the sidebar)'], ['Esc', 'Close dialog'], ['Tab / Shift+Tab', 'Move focus']];

export function SettingsView() {
  const s = useSettings();
  const { status, check } = useEngine();
  return (
    <>
      <PageHeader title="Settings" subtitle="Preferences are stored locally on this device." />
      <div className="grid gap-5 lg:grid-cols-2">
        <Panel title="Appearance">
          <div className="space-y-5">
            <div><Label>Theme</Label><Segmented<ThemeMode> label="Theme" value={s.mode} onChange={s.setMode} options={[{ value: 'dark', label: 'Dark' }, { value: 'light', label: 'Light' }]} /></div>
            <div><Label>Table density</Label><Segmented<Density> label="Density" value={s.density} onChange={s.setDensity} options={[{ value: 'comfortable', label: 'Comfortable' }, { value: 'compact', label: 'Compact' }]} /></div>
            <div>
              <Label htmlFor="font-size" hint={`${s.fontSize}px`}>Font size</Label>
              <input id="font-size" type="range" min={12} max={16} step={1} value={s.fontSize} onChange={(e) => s.setFontSize(Number(e.target.value))} className="w-full accent-[var(--text-primary)]" />
            </div>
          </div>
        </Panel>
        <Panel title="Typography">
          <p className="text-xs text-fg-soft">Interface and technical values use the Mono family.</p>
          <p className="mt-3 break-words font-mono text-[11px] text-fg-muted">Active stack: JetBrains Mono → IBM Plex Mono → Cascadia Mono → SF Mono → system monospace</p>
          <p className="mt-3 text-[11px] text-fg-muted">Bundled font files and a font picker will appear here once the font collection is supplied.</p>
        </Panel>
        <Panel title="Analyst identity">
          <Label htmlFor="analyst" hint="recorded in custody and audit entries">Name</Label>
          <Input id="analyst" value={s.analyst} maxLength={64} placeholder="local-analyst" onChange={(e) => s.setAnalyst(e.target.value)} />
        </Panel>
        <Panel title="Engine" actions={<Button size="sm" onClick={() => void check()}>Test</Button>}>
          <dl className="space-y-2 text-xs">
            <div className="flex justify-between gap-4"><dt className="text-fg-muted">Endpoint</dt><dd className="font-mono">{engineUrl()}</dd></div>
            <div className="flex justify-between gap-4"><dt className="text-fg-muted">Status</dt><dd>{status === 'online' ? <Badge tone="outline">● Online</Badge> : status === 'offline' ? <Badge tone="alert">▲ Offline</Badge> : <Badge>… Checking</Badge>}</dd></div>
          </dl>
        </Panel>
        <Panel title="Keyboard shortcuts">
          <dl className="space-y-2 text-xs">{SHORTCUTS.map(([k, d]) => <div key={k} className="flex justify-between gap-4"><dt className="font-mono">{k}</dt><dd className="text-fg-muted">{d}</dd></div>)}</dl>
        </Panel>
      </div>
    </>
  );
}
