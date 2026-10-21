import logo from '@/assets/xolograph.svg';
import pkg from '../../../package.json';
import { NAV_FLAT, NAV_GROUPS } from '@/app/navigation';
import { Badge } from '@/components/ui';
import { useNav } from '@/state/navStore';

export function Sidebar() {
  const { view, go } = useNav();
  return (
    <aside className="flex h-full w-60 shrink-0 flex-col border-r border-line bg-raised">
      <div className="flex h-14 items-center gap-3 border-b border-line px-4">
        <span className="flex h-8 w-8 items-center justify-center bg-black p-1"><img src={logo} alt="" className="h-full w-full" /></span>
        <span className="text-sm font-bold uppercase tracking-[0.28em]">Xolograph</span>
      </div>
      <nav aria-label="Primary" className="flex-1 overflow-y-auto py-3">
        {NAV_GROUPS.map((g) => (
          <div key={g.title} className="mb-4">
            <div className="px-4 pb-1.5 text-[10px] uppercase tracking-[0.18em] text-fg-muted">{g.title}</div>
            <ul>
              {g.items.map((item) => {
                const active = view === item.id;
                const n = NAV_FLAT.indexOf(item) + 1;
                return (
                  <li key={item.id}>
                    <button
                      type="button"
                      onClick={() => go(item.id)}
                      aria-current={active ? 'page' : undefined}
                      title={`Alt+${n}`}
                      className={`flex w-full items-center gap-3 border-l-2 px-4 py-2 text-left text-xs transition-colors ${
                        active ? 'border-fg bg-selected font-semibold text-fg' : 'border-transparent text-fg-soft hover:bg-hover hover:text-fg'
                      }`}
                    >
                      <item.icon size={15} aria-hidden />
                      <span className="flex-1 truncate">{item.label}</span>
                      {item.planned && <Badge>Soon</Badge>}
                    </button>
                  </li>
                );
              })}
            </ul>
          </div>
        ))}
      </nav>
      <div className="border-t border-line px-4 py-3 text-[10px] uppercase tracking-wider text-fg-muted">
        v{pkg.version} · local-first
      </div>
    </aside>
  );
}
