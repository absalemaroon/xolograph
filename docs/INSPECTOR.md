# File and Metadata Inspector (Phase 7)

Tools > **File Inspector**. Inspects a local file or an evidence-vault item. Processing is local and read-only: the file is
streamed once for hashing, at most 64 MiB are held in memory for metadata parsing, and nothing is uploaded, rendered,
executed or extracted. Parsers are written against the Go standard library only (no third-party parsing dependencies to vet),
bounds-checked, and wrapped so a malformed file produces a warning instead of a crash.

## What it reports
- **Type, from content, not name**: magic-byte signatures (images, PDF, Office/OpenDocument, archives, executables, media, SQLite, text kinds). A name that disagrees with the content is flagged.
- **Hashes**: SHA-256 and SHA-512 (primary) plus SHA-1 and MD5 (legacy, labelled as such), computed over the whole file.
- **Integrity comparison**: against an expected MD5/SHA-1/SHA-256/SHA-512 you enter, against the digest recorded in the vault (vault items), or against another file inspected earlier (baseline).
- **Images**: format and dimensions; JPEG EXIF (camera, software, timestamps, orientation, serial numbers, lens, GPS), XMP highlights, PNG text chunks/`tIME`/`eXIf`, embedded-thumbnail notice.
- **PDF**: version, Info dictionary (title, author, producer, dates) read from the trailer's Info object only, revisions, approximate page objects, and active-content markers (JavaScript, OpenAction, Launch, embedded files).
- **Office / OpenDocument**: author, last-modified-by, company, dates, application; macro and external-link markers.
- **Archives**: entry count, declared size, traversal and decompression-ratio warnings, first entries listed (never extracted).
- **Text**: encoding, line count, JSON/CSV/HTML/SVG/XML/script recognition and active-content warnings.
- **Export** as JSON or a plain-text report (limitations are included in both), or **attach to a case** as a finding with the full report.

## Rules and limits
- Metadata is self-declared and never presented as proof of origin, authenticity, time or place. Every report carries that statement.
- GPS coordinates are shown as numbers only (no map tiles, so nothing leaks to a map service) and trigger a *sensitive* warning, as do device serial numbers, owner/author names, company names and PDF authors.
- Not supported (reported as such, hashes still computed): HEIC/WebP/TIFF/video metadata, legacy `.doc/.xls` properties, compressed PDF object streams, encrypted content, MakerNotes.
- A PDF whose Info dictionary lives in a compressed object stream shows "no readable Info" rather than guessing.
- Files over 8 GiB are refused; for files over 64 MiB only the first 64 MiB are examined for metadata.
- Inspecting is an analysis aid and is not itself logged in the custody chain; attaching a result to a case is logged as a finding.
