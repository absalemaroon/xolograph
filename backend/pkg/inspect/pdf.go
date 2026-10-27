package inspect

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
)

var (
	pdfInfoRefRe = regexp.MustCompile(`/Info\s+(\d+)\s+(\d+)\s+R`)
	pdfPageRe    = regexp.MustCompile(`/Type\s*/Page[^s]`)
	pdfDateRe    = regexp.MustCompile(`^D:(\d{4})(\d{2})?(\d{2})?(\d{2})?(\d{2})?(\d{2})?([Zz+-])?(\d{2})?'?(\d{2})?`)
)

var pdfInfoKeys = []string{"Title", "Author", "Subject", "Keywords", "Creator", "Producer", "CreationDate", "ModDate"}

// parsePDF reads the document Info dictionary (when it is stored in plain text), flags active-content markers and
// notes incremental updates. Nothing is rendered or executed. The scan is textual and therefore best-effort.
func parsePDF(r *Report, b []byte) {
	if i := bytes.Index(b[:min(len(b), 1024)], []byte("%PDF-")); i > 0 {
		r.warn("caution", "pdf_header_offset", fmt.Sprintf("The PDF header is not at the start of the file (found at byte %d). Data before it is unusual and can indicate a polyglot or disguised file.", i))
	}
	if i := bytes.Index(b[:min(len(b), 1024)], []byte("%PDF-")); i >= 0 && len(b) >= i+8 {
		r.add("PDF", "Version", string(b[i+5:i+8]))
	}
	encrypted := bytes.Contains(b, []byte("/Encrypt"))
	if encrypted {
		r.add("PDF", "Encrypted", "yes — document information may be unreadable here")
		r.warn("info", "pdf_encrypted", "The PDF declares encryption; properties inside encrypted structures cannot be read by this inspector.")
	}
	if rev := bytes.Count(b, []byte("%%EOF")); rev > 1 {
		r.add("PDF", "Revisions", fmt.Sprintf("%d end-of-file markers (incremental updates or appended data)", rev))
		r.warn("info", "pdf_revisions", "The file contains more than one %EOF marker: it was updated incrementally or has data appended. Earlier revisions may still be present in the file.")
	}
	if pages := len(pdfPageRe.FindAll(b, -1)); pages > 0 {
		r.add("PDF", "Page objects (approximate)", strconv.Itoa(pages))
	}

	found := false
	if ms := pdfInfoRefRe.FindAllSubmatch(b, -1); len(ms) > 0 {
		m := ms[len(ms)-1] // the last trailer wins when the file was updated
		objRe := regexp.MustCompile(`(?:^|[\r\n\s])` + regexp.QuoteMeta(string(m[1])) + `\s+` + regexp.QuoteMeta(string(m[2])) + `\s+obj`)
		locs := objRe.FindAllIndex(b, -1)
		if len(locs) > 0 {
			start := locs[len(locs)-1][1]
			end := bytes.Index(b[start:], []byte("endobj"))
			if end < 0 {
				end = len(b) - start
			}
			body := b[start : start+end]
			for _, k := range pdfInfoKeys {
				if v, ok := pdfStringValue(body, k); ok {
					found = true
					if strings.HasSuffix(k, "Date") {
						v = pdfDate(v)
					}
					r.add("PDF info", k, v)
					switch k {
					case "Author":
						r.warn("sensitive", "pdf_author", "The PDF names an author, which may identify a person or organisation.")
					}
				}
			}
		}
	}
	if !found {
		r.warn("info", "pdf_no_info", "No readable document Info dictionary was found. It may be absent, stripped, encrypted, or stored in a compressed object stream this inspector does not decompress.")
	}
	if bytes.Contains(b, []byte("<x:xmpmeta")) {
		r.add("PDF", "XMP metadata", "present (not decoded)")
	}
	for _, f := range []struct{ pat, code, msg string }{
		{"/JavaScript", "pdf_js", "The PDF contains JavaScript markers."},
		{"/JS ", "pdf_js", "The PDF contains JavaScript markers."},
		{"/OpenAction", "pdf_openaction", "The PDF defines an action that runs when it is opened."},
		{"/Launch", "pdf_launch", "The PDF contains a Launch action, which can start other programs."},
		{"/EmbeddedFile", "pdf_embedded", "The PDF contains embedded files."},
	} {
		if bytes.Contains(b, []byte(f.pat)) && !hasWarn(r, f.code) {
			r.warn("caution", f.code, f.msg+" Active or embedded content is a common malware vector; this inspector did not execute or extract it.")
		}
	}
}

func hasWarn(r *Report, code string) bool {
	for _, w := range r.Warnings {
		if w.Code == code {
			return true
		}
	}
	return false
}

// pdfStringValue finds /Key followed by a literal ( … ) or hexadecimal < … > string inside one object body.
func pdfStringValue(body []byte, key string) (string, bool) {
	re := regexp.MustCompile(`/` + key + `\s*([(<])`)
	loc := re.FindSubmatchIndex(body)
	if loc == nil {
		return "", false
	}
	i := loc[3] // index just after the opening delimiter
	if body[loc[2]] == '<' {
		end := bytes.IndexByte(body[i:], '>')
		if end < 0 || end > 4096 {
			return "", false
		}
		return decodePDFText(unhex(body[i : i+end])), true
	}
	var out []byte
	depth := 1
	for ; i < len(body) && len(out) < 4096; i++ {
		c := body[i]
		switch {
		case c == '\\' && i+1 < len(body):
			i++
			switch n := body[i]; {
			case n == 'n':
				out = append(out, '\n')
			case n == 'r':
				out = append(out, '\r')
			case n == 't':
				out = append(out, '\t')
			case n >= '0' && n <= '7':
				v := int(n - '0')
				for k := 0; k < 2 && i+1 < len(body) && body[i+1] >= '0' && body[i+1] <= '7'; k++ {
					i++
					v = v*8 + int(body[i]-'0')
				}
				out = append(out, byte(v))
			case n == '\n' || n == '\r':
			default:
				out = append(out, n)
			}
		case c == '(':
			depth++
			out = append(out, c)
		case c == ')':
			depth--
			if depth == 0 {
				return decodePDFText(out), true
			}
			out = append(out, c)
		default:
			out = append(out, c)
		}
	}
	return "", false
}

func unhex(h []byte) []byte {
	var clean []byte
	for _, c := range h {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			clean = append(clean, c)
		}
	}
	if len(clean)%2 == 1 {
		clean = append(clean, '0')
	}
	out := make([]byte, 0, len(clean)/2)
	for i := 0; i+1 < len(clean); i += 2 {
		v, _ := strconv.ParseUint(string(clean[i:i+2]), 16, 8)
		out = append(out, byte(v))
	}
	return out
}

// decodePDFText handles UTF-16BE (BOM FE FF) and treats everything else as Latin-1-compatible bytes.
func decodePDFText(b []byte) string {
	if len(b) >= 2 && b[0] == 0xFE && b[1] == 0xFF {
		u := make([]uint16, 0, len(b)/2)
		for i := 2; i+1 < len(b); i += 2 {
			u = append(u, uint16(b[i])<<8|uint16(b[i+1]))
		}
		return string(utf16.Decode(u))
	}
	rs := make([]rune, len(b))
	for i, c := range b {
		rs[i] = rune(c)
	}
	return string(rs)
}

// pdfDate renders a PDF date (D:YYYYMMDDHHmmSS+HH'mm') readably while keeping the original.
func pdfDate(v string) string {
	m := pdfDateRe.FindStringSubmatch(strings.TrimSpace(v))
	if m == nil {
		return v
	}
	get := func(i int, def string) string {
		if m[i] == "" {
			return def
		}
		return m[i]
	}
	zone := "(no timezone stated)"
	switch m[7] {
	case "Z", "z":
		zone = "UTC"
	case "+", "-":
		zone = m[7] + get(8, "00") + ":" + get(9, "00")
	}
	return fmt.Sprintf("%s-%s-%s %s:%s:%s %s  [raw %s]", m[1], get(2, "01"), get(3, "01"), get(4, "00"), get(5, "00"), get(6, "00"), zone, v)
}
