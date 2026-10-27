import type { InspectionReport } from '@/api/types';

const LEVEL: Record<string, string> = { sensitive: 'SENSITIVE', caution: 'CAUTION', info: 'NOTE' };

/** A plain-text report that carries the limitations with it, so an exported copy is never read as proof. */
export function reportToText(r: InspectionReport): string {
  const L: string[] = [];
  L.push('XOLOGRAPH FILE INSPECTION REPORT', '='.repeat(32), '');
  L.push(`File:          ${r.name}`, `Size:          ${r.sizeBytes} bytes`, `Detected type: ${r.type.label} (${r.type.mime})`, `Type basis:    ${r.type.basis}`);
  if (r.type.declaredExtension) L.push(`Extension:     .${r.type.declaredExtension}${r.type.extensionMatches === false ? '  (DOES NOT MATCH CONTENT)' : ''}`);
  if (r.image) L.push(`Image:         ${r.image.format}, ${r.image.width} x ${r.image.height} px`);
  L.push(`Inspected:     ${r.inspectedAt} by ${r.toolVersion}`, '');
  L.push('HASHES', `  SHA-256  ${r.hashes.sha256}`, `  SHA-512  ${r.hashes.sha512}`, `  SHA-1    ${r.hashes.sha1}   (legacy; not collision resistant)`, `  MD5      ${r.hashes.md5}   (legacy; not collision resistant)`, '');
  if (r.integrity) L.push('INTEGRITY COMPARISON', `  ${r.integrity.algorithm}: ${r.integrity.match ? 'MATCH' : 'DOES NOT MATCH'}`, `  expected ${r.integrity.expected}`, `  actual   ${r.integrity.actual}`, '');
  if (r.warnings.length) {
    L.push('WARNINGS');
    for (const w of r.warnings) L.push(`  [${LEVEL[w.level] ?? w.level}] ${w.message}`);
    L.push('');
  }
  if (r.gps) L.push('GPS (self-declared, unverified)', `  ${r.gps.latitude}, ${r.gps.longitude}${r.gps.altitudeMeters !== undefined ? `, ${r.gps.altitudeMeters} m` : ''}`, '');
  if (r.metadata.length) {
    L.push('METADATA (self-declared, unverified)');
    let group = '';
    for (const f of r.metadata) {
      if (f.group !== group) { group = f.group; L.push(`  -- ${group}`); }
      L.push(`     ${f.name}: ${f.value}`);
    }
    L.push('');
  }
  if (r.truncated) L.push(`Note: only the first ${r.parsedBytes} bytes were examined for metadata; hashes cover the whole file.`, '');
  L.push('LIMITATIONS');
  for (const l of r.limitations) L.push(`  - ${l}`);
  return L.join('\n') + '\n';
}
