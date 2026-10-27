package inspect

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

func itoa(n int) string { return strconv.Itoa(n) }

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func humanBytes(n uint64) string {
	const k = 1024.0
	f := float64(n)
	switch {
	case f < k:
		return fmt.Sprintf("%d B", n)
	case f < k*k:
		return fmt.Sprintf("%.1f KiB", f/k)
	case f < k*k*k:
		return fmt.Sprintf("%.1f MiB", f/k/k)
	}
	return fmt.Sprintf("%.2f GiB", f/k/k/k)
}

// parseText reports encoding and size facts for text-like content, and flags active content in HTML/SVG/scripts.
func parseText(r *Report, b []byte) {
	enc := "UTF-8 (no byte-order mark)"
	switch {
	case bytes.HasPrefix(b, []byte("\xef\xbb\xbf")):
		enc = "UTF-8 with byte-order mark"
	case bytes.HasPrefix(b, []byte("\xff\xfe")):
		enc = "UTF-16 little-endian"
	case bytes.HasPrefix(b, []byte("\xfe\xff")):
		enc = "UTF-16 big-endian"
	case !utf8.Valid(b[:min(len(b), 64<<10)]):
		enc = "not valid UTF-8 (possibly a legacy 8-bit encoding)"
	}
	r.add("Text", "Encoding", enc)
	if !r.Truncated {
		r.add("Text", "Lines", itoa(bytes.Count(b, []byte("\n"))+1))
	}
	low := strings.ToLower(string(b[:min(len(b), 1<<20)]))
	switch r.Type.Kind {
	case "html", "svg":
		if strings.Contains(low, "<script") || strings.Contains(low, "onload=") || strings.Contains(low, "javascript:") {
			r.warn("caution", "active_content", "This "+r.Type.Label+" contains script or event-handler markup. Opening it in a browser could run code; it was not rendered here.")
		}
	case "script":
		r.add("Text", "Interpreter line", firstLine(b))
	case "xml":
		if strings.Contains(low, "<!entity") {
			r.warn("caution", "xml_entities", "The XML declares entities, which can be abused by some parsers (entity expansion or external file inclusion).")
		}
	}
}

func firstLine(b []byte) string {
	if i := bytes.IndexByte(b, '\n'); i >= 0 {
		b = b[:i]
	}
	return string(b[:min(len(b), 200)])
}
