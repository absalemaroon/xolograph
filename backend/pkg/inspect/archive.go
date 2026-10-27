package inspect

import (
	"archive/zip"
	"errors"
	"encoding/xml"
	"io"
	"os"
	"path"
	"strings"
)

const (
	maxZipEntries  = 20000
	maxMetaXMLSize = 1 << 20
	listEntries    = 25
)

// parseZip lists a ZIP container (without extracting anything) and reads document properties from Office Open XML and OpenDocument files.
func parseZip(r *Report, f *os.File, size int64) {
	zr, err := zip.NewReader(f, size)
	if err != nil && !(errors.Is(err, zip.ErrInsecurePath) && zr != nil) { // an insecure path still yields a usable reader
		zr = nil
	}
	if zr == nil {
		r.warn("caution", "zip_malformed", "The ZIP structure could not be read ("+clean(err.Error())+"). The file may be corrupt, truncated, or not a ZIP at all.")
		return
	}
	if len(zr.File) > maxZipEntries {
		r.warn("caution", "zip_many_entries", "The archive declares an unusually large number of entries; only a summary is shown.")
	}
	names := map[string]*zip.File{}
	var total uint64
	encrypted, traversal := 0, 0
	for i, e := range zr.File {
		if i >= maxZipEntries {
			break
		}
		names[e.Name] = e
		total += e.UncompressedSize64
		if e.Flags&1 != 0 {
			encrypted++
		}
		if strings.Contains(e.Name, "..") || strings.HasPrefix(e.Name, "/") || strings.Contains(e.Name, "\\") {
			traversal++
		}
	}
	r.add("Archive", "Entries", itoa(len(zr.File)))
	r.add("Archive", "Declared uncompressed size", humanBytes(total))
	if r.SizeBytes > 0 && total/uint64(r.SizeBytes) > 1000 {
		r.warn("caution", "zip_ratio", "The declared uncompressed size is more than 1000 times the file size, a pattern used by decompression bombs. Do not extract this archive without limits.")
	}
	if encrypted > 0 {
		r.warn("info", "zip_encrypted", itoa(encrypted)+" entr"+plural(encrypted, "y is", "ies are")+" encrypted; contents and some metadata cannot be read.")
	}
	if traversal > 0 {
		r.warn("caution", "zip_traversal", itoa(traversal)+" entry name(s) contain path-traversal or absolute-path patterns. Extracting such an archive naively can write files outside the target folder.")
	}

	switch {
	case names["[Content_Types].xml"] != nil && (hasPrefix(names, "word/") || hasPrefix(names, "xl/") || hasPrefix(names, "ppt/")):
		kind, label, mime := "docx", "Word document (Office Open XML)", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		switch {
		case hasPrefix(names, "xl/"):
			kind, label, mime = "xlsx", "Excel workbook (Office Open XML)", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		case hasPrefix(names, "ppt/"):
			kind, label, mime = "pptx", "PowerPoint presentation (Office Open XML)", "application/vnd.openxmlformats-officedocument.presentationml.presentation"
		}
		setOffice(r, kind, label, mime, []string{kind})
		readProps(r, names["docProps/core.xml"], "Document properties", map[string]string{
			"title": "Title", "subject": "Subject", "creator": "Author", "keywords": "Keywords", "description": "Comments",
			"lastModifiedBy": "Last modified by", "revision": "Revision", "created": "Created", "modified": "Modified", "category": "Category",
		}, []string{"Author", "Last modified by"})
		readProps(r, names["docProps/app.xml"], "Application properties", map[string]string{
			"Application": "Application", "AppVersion": "Application version", "Company": "Company", "Manager": "Manager", "Template": "Template", "TotalTime": "Total editing time (min)",
		}, []string{"Company", "Manager"})
		if names["docProps/custom.xml"] != nil {
			r.add("Document properties", "Custom properties", "present (not decoded)")
		}
		if hasSuffix(names, "vbaProject.bin") {
			r.warn("caution", "macros", "The document contains a VBA macro project. Macros are a common malware vector; this inspector did not read or run them.")
		}
		if hasPrefix(names, "xl/externalLinks/") || hasSubstr(names, "/_rels/") && hasSubstr(names, "externalLink") {
			r.warn("info", "external_links", "The document references external links or data sources.")
		}
	case names["mimetype"] != nil && names["meta.xml"] != nil:
		label, mime, extn := "OpenDocument file", "application/vnd.oasis.opendocument", "odt"
		if mt := readSmall(names["mimetype"]); strings.Contains(mt, "spreadsheet") {
			label, mime, extn = "OpenDocument spreadsheet", "application/vnd.oasis.opendocument.spreadsheet", "ods"
		} else if strings.Contains(mt, "presentation") {
			label, mime, extn = "OpenDocument presentation", "application/vnd.oasis.opendocument.presentation", "odp"
		} else if strings.Contains(mt, "text") {
			label, mime = "OpenDocument text", "application/vnd.oasis.opendocument.text"
		}
		setOffice(r, "odf", label, mime, []string{extn})
		readProps(r, names["meta.xml"], "Document properties", map[string]string{
			"title": "Title", "subject": "Subject", "creator": "Last modified by", "initial-creator": "Author", "creation-date": "Created", "date": "Modified",
			"generator": "Application", "keyword": "Keywords",
		}, []string{"Author", "Last modified by"})
	default:
		n := 0
		for i, e := range zr.File {
			if i >= listEntries {
				break
			}
			r.add("Archive contents", itoa(i+1), e.Name+" ("+humanBytes(e.UncompressedSize64)+")")
			n++
		}
		if len(zr.File) > n {
			r.add("Archive contents", "…", itoa(len(zr.File)-n)+" more entries not listed")
		}
		if hasSuffix(names, ".class") && names["META-INF/MANIFEST.MF"] != nil {
			r.warn("caution", "jar", "This looks like a Java archive (JAR), which can contain executable code.")
		}
	}
}

func setOffice(r *Report, kind, label, mime string, exts []string) {
	r.Type.Kind, r.Type.Label, r.Type.MIME, r.Type.Category, r.Type.exts = "zip", label, mime, "document", exts
	r.Type.Basis = "file signature and package structure (" + kind + ")"
}

func readSmall(f *zip.File) string {
	if f == nil || f.UncompressedSize64 > maxMetaXMLSize {
		return ""
	}
	rc, err := f.Open()
	if err != nil {
		return ""
	}
	defer rc.Close()
	b, _ := io.ReadAll(io.LimitReader(rc, maxMetaXMLSize))
	return string(b)
}

// readProps collects the text of selected elements (matched by local name, namespaces ignored) from a small XML part.
func readProps(r *Report, f *zip.File, group string, want map[string]string, personal []string) {
	if f == nil {
		return
	}
	if f.UncompressedSize64 > maxMetaXMLSize {
		r.warn("info", "props_large", "A document-properties part is unusually large and was not read.")
		return
	}
	body := readSmall(f)
	dec := xml.NewDecoder(strings.NewReader(body))
	dec.Strict = false
	cur := ""
	seen := map[string]bool{}
	for tokens := 0; tokens < 5000; tokens++ {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			cur = t.Name.Local
		case xml.EndElement:
			cur = ""
		case xml.CharData:
			if label, ok := want[cur]; ok && !seen[label] {
				if v := clean(string(t)); v != "" {
					seen[label] = true
					r.add(group, label, v)
					for _, p := range personal {
						if p == label {
							if !hasWarn(r, "doc_personal") {
								r.warn("sensitive", "doc_personal", "The document properties name an author, editor, company or manager. Sharing the file would disclose these names.")
							}
						}
					}
				}
			}
		}
	}
}

func hasPrefix(m map[string]*zip.File, p string) bool {
	for k := range m {
		if strings.HasPrefix(k, p) {
			return true
		}
	}
	return false
}

func hasSuffix(m map[string]*zip.File, s string) bool {
	for k := range m {
		if strings.HasSuffix(k, s) || path.Base(k) == s {
			return true
		}
	}
	return false
}

func hasSubstr(m map[string]*zip.File, s string) bool {
	for k := range m {
		if strings.Contains(k, s) {
			return true
		}
	}
	return false
}
