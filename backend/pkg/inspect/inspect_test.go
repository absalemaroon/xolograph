package inspect

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func inspectFile(t *testing.T, name string, data []byte, expected string) *Report {
	t.Helper()
	r, err := File(context.Background(), write(t, name, data), expected)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return r
}

func field(r *Report, group, name string) string {
	for _, f := range r.Metadata {
		if f.Group == group && f.Name == name {
			return f.Value
		}
	}
	return ""
}

func hasCode(r *Report, code string) bool {
	for _, w := range r.Warnings {
		if w.Code == code {
			return true
		}
	}
	return false
}

func levelOf(r *Report, code string) string {
	for _, w := range r.Warnings {
		if w.Code == code {
			return w.Level
		}
	}
	return ""
}

// ---- builders

func ent(tag, typ uint16, count uint32, val uint32) []byte {
	b := make([]byte, 12)
	binary.LittleEndian.PutUint16(b[0:], tag)
	binary.LittleEndian.PutUint16(b[2:], typ)
	binary.LittleEndian.PutUint32(b[4:], count)
	binary.LittleEndian.PutUint32(b[8:], val)
	return b
}

func inline2(a, b byte) uint32 { return uint32(a) | uint32(b)<<8 }

func rationals(vals ...[2]uint32) []byte {
	var out []byte
	for _, v := range vals {
		out = binary.LittleEndian.AppendUint32(out, v[0])
		out = binary.LittleEndian.AppendUint32(out, v[1])
	}
	return out
}

// buildTIFF assembles a little-endian EXIF block with camera, serial, date and GPS (48°51'30"N 2°17'40"E).
func buildTIFF() []byte {
	var b []byte
	b = append(b, 'I', 'I', 42, 0, 8, 0, 0, 0)
	// IFD0 at 8: 5 entries (66 bytes) -> data starts at 158
	b = binary.LittleEndian.AppendUint16(b, 5)
	b = append(b, ent(0x010F, 2, 5, 158)...)
	b = append(b, ent(0x0110, 2, 5, 163)...)
	b = append(b, ent(0x0112, 3, 1, 6)...)
	b = append(b, ent(0x8769, 4, 1, 74)...)
	b = append(b, ent(0x8825, 4, 1, 104)...)
	b = append(b, 0, 0, 0, 0)
	// Exif IFD at 74: 2 entries (30 bytes)
	b = binary.LittleEndian.AppendUint16(b, 2)
	b = append(b, ent(0x9003, 2, 20, 168)...)
	b = append(b, ent(0xA431, 2, 6, 188)...)
	b = append(b, 0, 0, 0, 0)
	// GPS IFD at 104: 4 entries (54 bytes)
	b = binary.LittleEndian.AppendUint16(b, 4)
	b = append(b, ent(1, 2, 2, inline2('N', 0))...)
	b = append(b, ent(2, 5, 3, 194)...)
	b = append(b, ent(3, 2, 2, inline2('E', 0))...)
	b = append(b, ent(4, 5, 3, 218)...)
	b = append(b, 0, 0, 0, 0)
	b = append(b, "Acme\x00"...)
	b = append(b, "Cam1\x00"...)
	b = append(b, "2020:01:02 03:04:05\x00"...)
	b = append(b, "SN123\x00"...)
	b = append(b, rationals([2]uint32{48, 1}, [2]uint32{51, 1}, [2]uint32{30, 1})...)
	b = append(b, rationals([2]uint32{2, 1}, [2]uint32{17, 1}, [2]uint32{40, 1})...)
	return b
}

func jpegWith(t *testing.T, app1 []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 8, 6)), nil); err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	if app1 == nil {
		return b
	}
	seg := append([]byte{0xFF, 0xE1, byte((len(app1) + 2) >> 8), byte(len(app1) + 2)}, app1...)
	return append(append(append([]byte{}, b[:2]...), seg...), b[2:]...)
}

func exifJPEG(t *testing.T) []byte { return jpegWith(t, append([]byte("Exif\x00\x00"), buildTIFF()...)) }

func pngWithText(t *testing.T, key, val string) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 3))); err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	data := append(append([]byte(key), 0), val...)
	chunk := binary.BigEndian.AppendUint32(nil, uint32(len(data)))
	chunk = append(chunk, "tEXt"...)
	chunk = append(chunk, data...)
	chunk = binary.BigEndian.AppendUint32(chunk, crc32.ChecksumIEEE(append([]byte("tEXt"), data...)))
	cut := len(b) - 12 // before IEND
	return append(append(append([]byte{}, b[:cut]...), chunk...), b[cut:]...)
}

func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// ---- tests

func TestHashesAndIntegrity(t *testing.T) {
	const sha = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	r := inspectFile(t, "note.txt", []byte("hello"), "sha256:"+strings.ToUpper(sha))
	if r.Hashes.SHA256 != sha || r.Hashes.MD5 != "5d41402abc4b2a76b9719d911017c592" || r.Hashes.SHA1 != "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d" || r.SizeBytes != 5 {
		t.Fatalf("hashes: %+v", r.Hashes)
	}
	if r.Integrity == nil || !r.Integrity.Match || r.Integrity.Algorithm != "SHA256" {
		t.Fatalf("integrity: %+v", r.Integrity)
	}
	if r := inspectFile(t, "note.txt", []byte("hello"), "5D41402ABC4B2A76B9719D911017C592"); r.Integrity == nil || !r.Integrity.Match || r.Integrity.Algorithm != "MD5" {
		t.Fatalf("md5 comparison: %+v", r.Integrity)
	}
	if r := inspectFile(t, "note.txt", []byte("hello!"), sha); r.Integrity == nil || r.Integrity.Match {
		t.Fatalf("a different file must not match: %+v", r.Integrity)
	}
	for _, bad := range []string{"xyz", strings.Repeat("a", 10), "sha999:" + sha} {
		if _, err := File(context.Background(), write(t, "a.txt", []byte("x")), bad); !errors.Is(err, ErrInvalid) {
			t.Fatalf("bad expected hash %q accepted: %v", bad, err)
		}
	}
	if len(r.Limitations) < 3 {
		t.Fatal("limitations must always accompany a report")
	}
}

func TestJPEGExifGPSAndSensitiveWarnings(t *testing.T) {
	r := inspectFile(t, "photo.jpg", exifJPEG(t), "")
	if r.Type.Kind != "jpeg" || r.Image == nil || r.Image.Width != 8 || r.Image.Height != 6 || r.Image.Format != "jpeg" {
		t.Fatalf("type/dims: %+v %+v", r.Type, r.Image)
	}
	if field(r, "EXIF", "Camera make") != "Acme" || field(r, "EXIF", "Camera model") != "Cam1" || field(r, "EXIF", "Date/time original") != "2020:01:02 03:04:05" || field(r, "EXIF", "Orientation") != "Rotated 90° CW" {
		t.Fatalf("exif fields: %+v", r.Metadata)
	}
	if r.GPS == nil || r.GPS.Latitude != 48.858333 || r.GPS.Longitude != 2.294444 {
		t.Fatalf("gps: %+v", r.GPS)
	}
	if levelOf(r, "gps") != "sensitive" || levelOf(r, "device_serial") != "sensitive" {
		t.Fatalf("sensitive-data warnings missing: %+v", r.Warnings)
	}
	if r.Type.ExtensionMatches == nil || !*r.Type.ExtensionMatches {
		t.Fatal("jpg must match JPEG content")
	}
}

func TestMalformedAndTruncatedFilesFailSafely(t *testing.T) {
	good := exifJPEG(t)
	// truncated inside the EXIF segment
	r := inspectFile(t, "cut.jpg", good[:40], "")
	if !hasCode(r, "jpeg_truncated") && !hasCode(r, "image_header") {
		t.Fatalf("truncation must be reported: %+v", r.Warnings)
	}
	// EXIF whose offsets point far outside the block
	evil := append([]byte("Exif\x00\x00"), 'I', 'I', 42, 0, 0xF0, 0xFF, 0xFF, 0x7F, 1, 0)
	r = inspectFile(t, "evil.jpg", jpegWith(t, evil), "")
	if r.GPS != nil || len(r.Warnings) == 0 && len(r.Metadata) != 0 {
		t.Fatalf("hostile offsets must yield nothing and no crash: %+v", r)
	}
	// IFD that points back to itself must not loop
	loop := []byte{'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x69, 0x87, 4, 0, 1, 0, 0, 0, 8, 0, 0, 0, 0, 0, 0, 0}
	inspectFile(t, "loop.jpg", jpegWith(t, append([]byte("Exif\x00\x00"), loop...)), "")
	// random bytes with a PNG signature, a PDF header and a ZIP header
	for _, name := range []string{"a.png", "b.pdf", "c.zip", "d.docx"} {
		sig := map[string]string{"a.png": "\x89PNG\r\n\x1a\n", "b.pdf": "%PDF-1.7\n", "c.zip": "PK\x03\x04", "d.docx": "PK\x03\x04"}[name]
		data := append([]byte(sig), bytes.Repeat([]byte{0xFF, 0x00, 0x7F, 0x80}, 500)...)
		r := inspectFile(t, name, data, "")
		if r.Hashes.SHA256 == "" {
			t.Fatalf("%s: hashing must still work on damaged files", name)
		}
	}
	// empty file, directory and missing file
	if r := inspectFile(t, "empty.bin", nil, ""); r.Type.Kind != "empty" || !hasCode(r, "empty") {
		t.Fatalf("empty: %+v", r.Type)
	}
	if _, err := File(context.Background(), t.TempDir(), ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("directory: %v", err)
	}
	if _, err := File(context.Background(), filepath.Join(t.TempDir(), "nope"), ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing: %v", err)
	}
}

func TestPNGTextChunksAndSpoofedExtension(t *testing.T) {
	data := pngWithText(t, "Author", "Jane Doe")
	r := inspectFile(t, "shot.png", data, "")
	if r.Type.Kind != "png" || r.Image == nil || r.Image.Width != 4 || field(r, "PNG text", "Author") != "Jane Doe" || levelOf(r, "png_author") != "sensitive" {
		t.Fatalf("png: %+v %+v", r.Type, r.Metadata)
	}
	spoof := inspectFile(t, "invoice.pdf", data, "")
	if spoof.Type.Kind != "png" || spoof.Type.ExtensionMatches == nil || *spoof.Type.ExtensionMatches || !hasCode(spoof, "extension_mismatch") {
		t.Fatalf("a PNG named .pdf must be flagged: %+v %+v", spoof.Type, spoof.Warnings)
	}
}

func TestPDFInfoAndActiveContent(t *testing.T) {
	pdf := "%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /Title (not the info title) >>\nendobj\n" +
		"4 0 obj\n<< /Title (Quarterly \\(draft\\) report) /Author (Jane Doe) /Producer <FEFF005000440046> /CreationDate (D:20200102030405+01'00') /OpenAction 3 0 R >>\nendobj\n" +
		"trailer\n<< /Root 1 0 R /Info 4 0 R >>\n%%EOF\n"
	r := inspectFile(t, "report.pdf", []byte(pdf), "")
	if r.Type.Kind != "pdf" || field(r, "PDF info", "Title") != "Quarterly (draft) report" || field(r, "PDF info", "Author") != "Jane Doe" || field(r, "PDF info", "Producer") != "PDF" {
		t.Fatalf("pdf info (must come from the Info object only): %+v", r.Metadata)
	}
	if d := field(r, "PDF info", "CreationDate"); !strings.HasPrefix(d, "2020-01-02 03:04:05 +01:00") {
		t.Fatalf("date: %q", d)
	}
	if field(r, "PDF", "Page objects (approximate)") != "1" || !hasCode(r, "pdf_openaction") || levelOf(r, "pdf_author") != "sensitive" {
		t.Fatalf("pdf flags: %+v %+v", r.Metadata, r.Warnings)
	}
	none := inspectFile(t, "bare.pdf", []byte("%PDF-1.4\n1 0 obj\n<< >>\nendobj\n%%EOF\n"), "")
	if !hasCode(none, "pdf_no_info") {
		t.Fatalf("missing info must be said, not silently empty: %+v", none.Warnings)
	}
	offset := inspectFile(t, "poly.pdf", append([]byte("GARBAGE"), pdf...), "")
	if !hasCode(offset, "pdf_header_offset") {
		t.Fatalf("polyglot header offset: %+v", offset.Warnings)
	}
}

func TestOfficeDocumentProperties(t *testing.T) {
	core := `<?xml version="1.0"?><cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/"><dc:title>Plan</dc:title><dc:creator>Jane Doe</dc:creator><cp:lastModifiedBy>John Roe</cp:lastModifiedBy><dcterms:created>2020-01-02T03:04:05Z</dcterms:created></cp:coreProperties>`
	app := `<Properties><Application>Word</Application><Company>Acme Ltd</Company></Properties>`
	files := map[string]string{"[Content_Types].xml": "<Types/>", "word/document.xml": "<w/>", "docProps/core.xml": core, "docProps/app.xml": app}
	r := inspectFile(t, "memo.docx", makeZip(t, files), "")
	if !strings.Contains(r.Type.Label, "Word document") || r.Type.ExtensionMatches == nil || !*r.Type.ExtensionMatches {
		t.Fatalf("type: %+v", r.Type)
	}
	if field(r, "Document properties", "Author") != "Jane Doe" || field(r, "Document properties", "Last modified by") != "John Roe" || field(r, "Application properties", "Company") != "Acme Ltd" || field(r, "Document properties", "Created") != "2020-01-02T03:04:05Z" {
		t.Fatalf("props: %+v", r.Metadata)
	}
	if levelOf(r, "doc_personal") != "sensitive" || hasCode(r, "macros") {
		t.Fatalf("warnings: %+v", r.Warnings)
	}
	files["word/vbaProject.bin"] = "x"
	if m := inspectFile(t, "memo.docx", makeZip(t, files), ""); !hasCode(m, "macros") {
		t.Fatalf("macro project must be flagged: %+v", m.Warnings)
	}
	plain := inspectFile(t, "bundle.zip", makeZip(t, map[string]string{"../evil.txt": "x", "a/b.txt": "y"}), "")
	if plain.Type.Kind != "zip" || !hasCode(plain, "zip_traversal") || field(plain, "Archive", "Entries") != "2" {
		t.Fatalf("zip listing/traversal: %+v %+v", plain.Metadata, plain.Warnings)
	}
}

func TestExecutablesAndTextKinds(t *testing.T) {
	pe := make([]byte, 0x80)
	copy(pe, "MZ")
	binary.LittleEndian.PutUint32(pe[0x3c:], 0x40)
	copy(pe[0x40:], "PE\x00\x00")
	r := inspectFile(t, "invoice.pdf", pe, "")
	if r.Type.Kind != "pe" || r.Type.Category != "executable" || !hasCode(r, "executable") || !hasCode(r, "extension_mismatch") {
		t.Fatalf("PE disguised as PDF: %+v %+v", r.Type, r.Warnings)
	}
	if r := inspectFile(t, "run.sh", []byte("#!/bin/sh\necho hi\n"), ""); r.Type.Kind != "script" || !hasCode(r, "executable") {
		t.Fatalf("script: %+v", r.Type)
	}
	if r := inspectFile(t, "benign.txt", []byte("click me"), ""); r.Type.Category != "text" || r.Type.ExtensionMatches == nil || !*r.Type.ExtensionMatches {
		t.Fatalf("plain text: %+v", r.Type)
	}
	if r := inspectFile(t, "x.js", []byte("alert(1)\n"), ""); r.Type.Category != "executable" || !hasCode(r, "script_extension") {
		t.Fatalf("js by extension: %+v", r.Type)
	}
	if r := inspectFile(t, "data.json", []byte(`{"a":[1,2]}`), ""); r.Type.Kind != "json" {
		t.Fatalf("json: %+v", r.Type)
	}
	if r := inspectFile(t, "t.csv", []byte("a,b,c\n1,2,3\n4,5,6\n7,8,9\n"), ""); r.Type.Kind != "csv" {
		t.Fatalf("csv: %+v", r.Type)
	}
	if r := inspectFile(t, "page.html", []byte("<!DOCTYPE html><html><script>x()</script></html>"), ""); r.Type.Kind != "html" || !hasCode(r, "active_content") {
		t.Fatalf("html: %+v %+v", r.Type, r.Warnings)
	}
	if r := inspectFile(t, "image.png", []byte("just text pretending"), ""); r.Type.ExtensionMatches == nil || *r.Type.ExtensionMatches || !hasCode(r, "extension_mismatch") {
		t.Fatalf("text named .png: %+v", r.Type)
	}
	if r := inspectFile(t, "blob.dat", bytes.Repeat([]byte{0x00, 0xFE, 0x13}, 300), ""); r.Type.Kind != "unknown" {
		t.Fatalf("unknown binary: %+v", r.Type)
	}
}

func TestCancelledContextStopsHashing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := File(ctx, write(t, "a.bin", []byte("data")), ""); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cancelled: %v", err)
	}
}
