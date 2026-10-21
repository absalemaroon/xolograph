// Builds the Go engine as a Tauri sidecar: src-tauri/binaries/xolographd-<target-triple>[.exe]
import { execFileSync } from 'node:child_process';
import { mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const backend = join(here, '..', 'backend');
const outDir = join(here, '..', 'src-tauri', 'binaries');

function hostTriple() {
  const out = execFileSync('rustc', ['-vV'], { encoding: 'utf8' });
  const line = out.split('\n').find((l) => l.startsWith('host:'));
  if (!line) throw new Error('Could not detect the Rust host target (is rustc installed?)');
  return line.split(':')[1].trim();
}

const triple = process.env.TAURI_ENV_TARGET_TRIPLE || hostTriple();
const goarch = triple.startsWith('x86_64') ? 'amd64' : triple.startsWith('aarch64') ? 'arm64' : null;
const goos = triple.includes('windows') ? 'windows' : triple.includes('apple-darwin') ? 'darwin' : triple.includes('linux') ? 'linux' : null;
if (!goarch || !goos) throw new Error(`Unsupported target triple: ${triple}`);

mkdirSync(outDir, { recursive: true });
const out = join(outDir, `xolographd-${triple}${goos === 'windows' ? '.exe' : ''}`);
console.log(`Building engine for ${triple} (${goos}/${goarch}) -> ${out}`);
execFileSync('go', ['build', '-trimpath', '-ldflags', '-s -w', '-o', out, './cmd/xolographd'], {
  cwd: backend,
  stdio: 'inherit',
  env: { ...process.env, CGO_ENABLED: '0', GOOS: goos, GOARCH: goarch },
});
