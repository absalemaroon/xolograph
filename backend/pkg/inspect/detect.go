package inspect

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type TypeInfo struct {
	Kind             string `json:"kind"`
	Label            string `json:"label"`
	MIME             string `json:"mime"`
	Category         string `json:"category"` // image | document | archive | executable | text | media | data | unknown
	Basis            string `json:"basis"`    // how the type was decided
	DeclaredExt      string `json:"declaredExtension"`
	ExtensionMatches *bool  `json:"extensionMatches,omitempty"`
	SniffedMIME      string `json:"sniffedMime,omitempty"`
	exts             []string
}

type magic struct {
	kind, label, mime, cat string
	exts                   []string
	test                   func(b []byte) bool
}

func prefix(p string) func([]byte) bool { return func(b []byte) bool { return bytes.HasPrefix(b, []byte(p)) } }

func at(b []byte, off int, s string) bool { return len(b) >= off+len(s) && string(b[off:off+len(s)]) == s }

var magics = []magic{
	{"png", "PNG image", "image/png", "image", []string{"png"}, prefix("\x89PNG\r\n\x1a\n")},
	{"jpeg", "JPEG image", "image/jpeg", "image", []string{"jpg", "jpeg", "jpe", "jfif"}, prefix("\xff\xd8\xff")},
	{"gif", "GIF image", "image/gif", "image", []string{"gif"}, func(b []byte) bool { return bytes.HasPrefix(b, []byte("GIF87a")) || bytes.HasPrefix(b, []byte("GIF89a")) }},
	{"webp", "WebP image", "image/webp", "image", []string{"webp"}, func(b []byte) bool { return at(b, 0, "RIFF") && at(b, 8, "WEBP") }},
	{"bmp", "BMP image", "image/bmp", "image", []string{"bmp", "dib"}, func(b []byte) bool { return len(b) >= 14 && at(b, 0, "BM") && b[6] == 0 && b[7] == 0 && b[8] == 0 && b[9] == 0 }},
	{"tiff", "TIFF image", "image/tiff", "image", []string{"tif", "tiff"}, func(b []byte) bool { return bytes.HasPrefix(b, []byte("II*\x00")) || bytes.HasPrefix(b, []byte("MM\x00*")) }},
	{"pdf", "PDF document", "application/pdf", "document", []string{"pdf"}, func(b []byte) bool { return bytes.Contains(b[:min(len(b), 1024)], []byte("%PDF-")) }},
	{"rtf", "Rich Text document", "application/rtf", "document", []string{"rtf"}, prefix("{\\rtf")},
	{"ole2", "Legacy Microsoft Office document (OLE2)", "application/x-ole-storage", "document", []string{"doc", "xls", "ppt", "msg"}, prefix("\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1")},
	{"zip", "ZIP archive", "application/zip", "archive", []string{"zip", "jar", "apk", "docx", "xlsx", "pptx", "odt", "ods", "odp", "epub"}, func(b []byte) bool { return bytes.HasPrefix(b, []byte("PK\x03\x04")) || bytes.HasPrefix(b, []byte("PK\x05\x06")) }},
	{"gzip", "gzip archive", "application/gzip", "archive", []string{"gz", "tgz"}, prefix("\x1f\x8b")},
	{"bzip2", "bzip2 archive", "application/x-bzip2", "archive", []string{"bz2"}, prefix("BZh")},
	{"xz", "xz archive", "application/x-xz", "archive", []string{"xz"}, prefix("\xfd7zXZ\x00")},
	{"7z", "7-Zip archive", "application/x-7z-compressed", "archive", []string{"7z"}, prefix("7z\xbc\xaf\x27\x1c")},
	{"rar", "RAR archive", "application/vnd.rar", "archive", []string{"rar"}, prefix("Rar!\x1a\x07")},
	{"tar", "tar archive", "application/x-tar", "archive", []string{"tar"}, func(b []byte) bool { return at(b, 257, "ustar") }},
	{"sqlite", "SQLite database", "application/vnd.sqlite3", "data", []string{"sqlite", "sqlite3", "db"}, prefix("SQLite format 3\x00")},
	{"elf", "ELF executable", "application/x-elf", "executable", []string{"elf", "so", "bin", "out", ""}, prefix("\x7fELF")},
	{"pe", "Windows executable (PE)", "application/vnd.microsoft.portable-executable", "executable", []string{"exe", "dll", "sys", "scr", "ocx"}, isPE},
	{"mz", "DOS/MZ executable", "application/x-dosexec", "executable", []string{"exe", "com", "dll"}, prefix("MZ")},
	{"class", "Java class file", "application/java-vm", "executable", []string{"class"}, isJavaClass},
	{"macho", "Mach-O executable", "application/x-mach-binary", "executable", []string{"", "dylib", "o", "bundle"}, isMachO},
	{"mp3", "MP3 audio", "audio/mpeg", "media", []string{"mp3"}, func(b []byte) bool { return bytes.HasPrefix(b, []byte("ID3")) || (len(b) > 2 && b[0] == 0xff && b[1]&0xe0 == 0xe0) }},
	{"wav", "WAV audio", "audio/wav", "media", []string{"wav"}, func(b []byte) bool { return at(b, 0, "RIFF") && at(b, 8, "WAVE") }},
	{"avi", "AVI video", "video/x-msvideo", "media", []string{"avi"}, func(b []byte) bool { return at(b, 0, "RIFF") && at(b, 8, "AVI ") }},
	{"ogg", "Ogg media", "application/ogg", "media", []string{"ogg", "oga", "ogv", "opus"}, prefix("OggS")},
	{"flac", "FLAC audio", "audio/flac", "media", []string{"flac"}, prefix("fLaC")},
	{"mkv", "Matroska/WebM media", "video/x-matroska", "media", []string{"mkv", "webm", "mka"}, prefix("\x1a\x45\xdf\xa3")},
	{"isobmff", "MP4/QuickTime/HEIF media", "video/mp4", "media", []string{"mp4", "m4a", "m4v", "mov", "heic", "heif", "avif", "3gp"}, func(b []byte) bool { return at(b, 4, "ftyp") }},
}

func isPE(b []byte) bool {
	if len(b) < 0x40 || !bytes.HasPrefix(b, []byte("MZ")) {
		return false
	}
	off := int(binary.LittleEndian.Uint32(b[0x3c:]))
	return off >= 0 && off <= len(b)-4 && at(b, off, "PE\x00\x00")
}

func isJavaClass(b []byte) bool {
	return len(b) >= 8 && bytes.HasPrefix(b, []byte{0xca, 0xfe, 0xba, 0xbe}) && binary.BigEndian.Uint32(b[4:]) >= 30
}

func isMachO(b []byte) bool {
	if len(b) < 8 {
		return false
	}
	if bytes.HasPrefix(b, []byte{0xca, 0xfe, 0xba, 0xbe}) {
		return binary.BigEndian.Uint32(b[4:]) < 30 // fat binary: a small architecture count
	}
	for _, m := range [][]byte{{0xfe, 0xed, 0xfa, 0xce}, {0xfe, 0xed, 0xfa, 0xcf}, {0xce, 0xfa, 0xed, 0xfe}, {0xcf, 0xfa, 0xed, 0xfe}} {
		if bytes.HasPrefix(b, m) {
			return true
		}
	}
	return false
}

var scriptExts = map[string]bool{"js": true, "vbs": true, "vbe": true, "ps1": true, "bat": true, "cmd": true, "sh": true, "lnk": true, "msi": true, "jar": true, "hta": true, "wsf": true, "scr": true, "py": true, "pl": true, "rb": true, "app": true, "command": true, "reg": true}
var binaryExts = map[string]bool{"png": true, "jpg": true, "jpeg": true, "gif": true, "pdf": true, "zip": true, "exe": true, "dll": true, "mp3": true, "mp4": true, "docx": true, "xlsx": true, "pptx": true, "7z": true, "rar": true, "gz": true, "webp": true, "bmp": true, "tif": true, "tiff": true, "sqlite": true, "db": true}

func ext(name string) string { return strings.ToLower(strings.TrimPrefix(filepath.Ext(name), ".")) }

func detectType(b []byte, name string) TypeInfo {
	t := TypeInfo{DeclaredExt: ext(name), Basis: "file signature (magic bytes)"}
	if len(b) > 0 {
		t.SniffedMIME = http.DetectContentType(b[:min(len(b), 512)])
	}
	for _, m := range magics {
		if m.test(b) {
			t.Kind, t.Label, t.MIME, t.Category, t.exts = m.kind, m.label, m.mime, m.cat, m.exts
			return t
		}
	}
	t.Basis = "content heuristics (no known signature)"
	head := b[:min(len(b), 64<<10)]
	switch {
	case len(b) == 0:
		t.Kind, t.Label, t.MIME, t.Category, t.Basis = "empty", "Empty file", "application/x-empty", "unknown", "file is empty"
	case bytes.HasPrefix(head, []byte("\xef\xbb\xbf")) || looksText(head) || bytes.HasPrefix(head, []byte("\xff\xfe")) || bytes.HasPrefix(head, []byte("\xfe\xff")):
		classifyText(&t, b, head)
	default:
		t.Kind, t.Label, t.MIME, t.Category, t.Basis = "unknown", "Unrecognised binary data", "application/octet-stream", "unknown", "no known signature; contains binary data"
	}
	return t
}

func looksText(b []byte) bool {
	if len(b) == 0 || bytes.IndexByte(b, 0) >= 0 {
		return false
	}
	valid := b
	for i := 0; i < 3 && len(valid) > 0 && !utf8.Valid(valid); i++ { // tolerate a cut multi-byte rune at the end
		valid = valid[:len(valid)-1]
	}
	if !utf8.Valid(valid) {
		return false
	}
	ctrl := 0
	for _, c := range valid {
		if c < 0x20 && c != '\n' && c != '\r' && c != '\t' && c != '\f' {
			ctrl++
		}
	}
	return ctrl*50 < len(valid)
}

func classifyText(t *TypeInfo, all, head []byte) {
	trim := bytes.TrimLeft(head, "\xef\xbb\xbf \t\r\n")
	low := strings.ToLower(string(trim[:min(len(trim), 512)]))
	set := func(kind, label, mime, cat string, exts ...string) { t.Kind, t.Label, t.MIME, t.Category, t.exts = kind, label, mime, cat, exts }
	switch {
	case strings.HasPrefix(low, "#!"):
		set("script", "Script (shebang)", "text/x-script", "executable", "sh", "py", "pl", "rb", "")
	case strings.HasPrefix(low, "<svg") || (strings.HasPrefix(low, "<?xml") && strings.Contains(low, "<svg")):
		set("svg", "SVG image (XML text)", "image/svg+xml", "image", "svg")
	case strings.HasPrefix(low, "<!doctype html") || strings.HasPrefix(low, "<html"):
		set("html", "HTML document", "text/html", "document", "html", "htm")
	case strings.HasPrefix(low, "<?xml"):
		set("xml", "XML document", "application/xml", "text", "xml")
	case (strings.HasPrefix(low, "{") || strings.HasPrefix(low, "[")) && len(all) <= 8<<20 && json.Valid(all):
		set("json", "JSON document", "application/json", "text", "json")
	case looksCSV(head):
		set("csv", "CSV table (likely)", "text/csv", "text", "csv", "tsv")
	default:
		set("text", "Plain text", "text/plain", "text", "txt", "log", "md", "ini", "cfg", "conf", "yml", "yaml", "csv", "tsv", "json", "xml", "")
	}
}

func looksCSV(b []byte) bool {
	lines := strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	if len(lines) < 4 {
		return false
	}
	for _, sep := range []string{",", ";", "\t"} {
		n := strings.Count(lines[0], sep)
		if n == 0 {
			continue
		}
		ok := 0
		for _, l := range lines[1:min(len(lines)-1, 6)] {
			if strings.Count(l, sep) == n {
				ok++
			}
		}
		if ok >= 2 {
			return true
		}
	}
	return false
}

// finishType decides whether the declared extension agrees with the detected content and adds the matching warnings.
func finishType(r *Report) {
	t := &r.Type
	e := t.DeclaredExt
	if e != "" && scriptExts[e] && t.Category != "executable" {
		t.Category = "executable"
		r.warn("caution", "script_extension", "The extension ."+e+" normally denotes an executable or script type, whatever the content looks like.")
		if t.Kind == "text" || t.Kind == "script" || t.Kind == "html" { // a script is expected to look like text
			return
		}
	}
	if t.Kind == "unknown" || t.Kind == "empty" || e == "" {
		return
	}
	match := false
	for _, x := range t.exts {
		if x == e {
			match = true
		}
	}
	if !match && t.Category == "text" && !binaryExts[e] && !scriptExts[e] {
		match = true
	}
	t.ExtensionMatches = &match
	if !match {
		r.warn("caution", "extension_mismatch", "The file is named ."+e+" but its content is "+t.Label+". A name that disagrees with the content is a common way to disguise a file; treat it with caution.")
	}
}
