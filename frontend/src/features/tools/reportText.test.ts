import { describe, expect, it } from 'vitest';
import type { InspectionReport } from '@/api/types';
import { reportToText } from './reportText';

const base: InspectionReport = {
  name: 'photo.jpg', sizeBytes: 10, hashes: { sha256: 'a'.repeat(64), sha512: 'b'.repeat(128), sha1: 'c'.repeat(40), md5: 'd'.repeat(32) },
  type: { kind: 'jpeg', label: 'JPEG image', mime: 'image/jpeg', category: 'image', basis: 'file signature (magic bytes)', declaredExtension: 'png', extensionMatches: false },
  metadata: [{ group: 'EXIF', name: 'Camera make', value: 'Acme' }], gps: { latitude: 1.5, longitude: -2.5 },
  integrity: { algorithm: 'SHA256', expected: 'x', actual: 'y', match: false },
  warnings: [{ level: 'sensitive', code: 'gps', message: 'Contains GPS.' }], limitations: ['Metadata is not proof.'],
  parsedBytes: 10, truncated: false, inspectedAt: '2026-01-01T00:00:00Z', toolVersion: 't/1',
};

describe('text report', () => {
  it('always carries limitations, warnings and mismatches', () => {
    const t = reportToText(base);
    expect(t).toContain('LIMITATIONS');
    expect(t).toContain('Metadata is not proof.');
    expect(t).toContain('[SENSITIVE] Contains GPS.');
    expect(t).toContain('DOES NOT MATCH CONTENT');
    expect(t).toContain('SHA256: DOES NOT MATCH');
    expect(t).toContain('self-declared, unverified');
  });
  it('omits empty sections', () => {
    const t = reportToText({ ...base, gps: undefined, integrity: undefined, warnings: [], metadata: [] });
    expect(t).not.toContain('WARNINGS');
    expect(t).not.toContain('GPS');
    expect(t).not.toContain('INTEGRITY');
  });
});
