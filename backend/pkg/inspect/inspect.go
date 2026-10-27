// Package inspect reads a local file and reports what can be determined about it without trusting it: content-based type,
// cryptographic hashes and embedded metadata. Files are opened read-only and are never executed, rendered or extracted.
// Metadata is self-declared by whatever produced the file; the report says so and never presents it as proof.
package inspect

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"image"
	_ "image/gif"  // decoders are registered for DecodeConfig (header only)
	_ "image/jpeg" // never used to decode pixels
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	MaxParseBytes = 64 << 20 // bytes of a file held in memory for metadata parsing
	MaxFileBytes  = 8 << 30  // larger files are refused
	maxPixels     = 100_000_000
)

var ErrInvalid = errors.New("invalid input")

type Field struct {
	Group string `json:"group"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Warning levels: "sensitive" (personal/location data that could be exposed if the file is shared), "caution" (risk or mismatch), "info".
type Warning struct {
	Level   string `json:"level"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Hashes struct {
	SHA256 string `json:"sha256"`
	SHA512 string `json:"sha512"`
	SHA1   string `json:"sha1"`
	MD5    string `json:"md5"`
}

type ImageInfo struct {
	Format string `json:"format"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type GPS struct {
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Altitude  *float64 `json:"altitudeMeters,omitempty"`
}

type Integrity struct {
	Algorithm string `json:"algorithm"`
	Expected  string `json:"expected"`
	Actual    string `json:"actual"`
	Match     bool   `json:"match"`
}

type Report struct {
	Name        string      `json:"name"`
	SizeBytes   int64       `json:"sizeBytes"`
	Hashes      Hashes      `json:"hashes"`
	Type        TypeInfo    `json:"type"`
	Image       *ImageInfo  `json:"image,omitempty"`
	Metadata    []Field     `json:"metadata"` // EXIF, XMP, PNG text, PDF info, document properties, archive listing
	GPS         *GPS        `json:"gps,omitempty"`
	Integrity   *Integrity  `json:"integrity,omitempty"`
	Warnings    []Warning   `json:"warnings"`
	Limitations []string    `json:"limitations"`
	ParsedBytes int64       `json:"parsedBytes"`
	Truncated   bool        `json:"truncated"`
	InspectedAt time.Time   `json:"inspectedAt"`
	ToolVersion string      `json:"toolVersion"`
}

func (r *Report) warn(level, code, msg string) { r.Warnings = append(r.Warnings, Warning{level, code, msg}) }
func (r *Report) add(group, name, value string) {
	value = clean(value)
	if value != "" {
		r.Metadata = append(r.Metadata, Field{group, name, value})
	}
}

// clean removes control characters and bounds length so hostile metadata cannot disturb a UI or report.
func clean(s string) string {
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return ' '
		}
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r < 0xa0) || r == 0xfffd {
			return -1
		}
		return r
	}, s)
	s = strings.TrimSpace(s)
	if len([]rune(s)) > 400 {
		s = string([]rune(s)[:400]) + "…"
	}
	return s
}

type capBuffer struct {
	buf []byte
	max int
}

func (c *capBuffer) Write(p []byte) (int, error) {
	if room := c.max - len(c.buf); room > 0 {
		if len(p) < room {
			room = len(p)
		}
		c.buf = append(c.buf, p[:room]...)
	}
	return len(p), nil
}

// ctxReader lets a long hash be cancelled when the request goes away.
type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}

// safe runs a parser and converts a panic (which malformed input must never be able to cause) into a warning.
func (r *Report) safe(name string, fn func()) {
	defer func() {
		if p := recover(); p != nil {
			r.warn("caution", "parser_failed", fmt.Sprintf("The %s parser stopped on unexpected content and its results are incomplete (%v).", name, p))
		}
	}()
	fn()
}

var hexRe = regexp.MustCompile(`^[0-9a-f]+$`)

// File inspects the file at path. expected may be a hex digest (MD5, SHA-1, SHA-256 or SHA-512, optionally prefixed "sha256:") to compare against.
func File(ctx context.Context, path, expected string) (*Report, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot read file: %v", ErrInvalid, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: not a regular file", ErrInvalid)
	}
	if info.Size() > MaxFileBytes {
		return nil, fmt.Errorf("%w: file is larger than the %d GiB limit", ErrInvalid, MaxFileBytes>>30)
	}
	want, algo, err := parseExpected(expected)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot open file: %v", ErrInvalid, err)
	}
	defer f.Close()

	hs := map[string]hash.Hash{"sha256": sha256.New(), "sha512": sha512.New(), "sha1": sha1.New(), "md5": md5.New()}
	head := &capBuffer{max: MaxParseBytes}
	n, err := io.Copy(io.MultiWriter(hs["sha256"], hs["sha512"], hs["sha1"], hs["md5"], head), ctxReader{ctx, f})
	if err != nil {
		return nil, fmt.Errorf("%w: read failed: %v", ErrInvalid, err)
	}
	sum := func(k string) string { return hex.EncodeToString(hs[k].Sum(nil)) }
	r := &Report{
		Name: filepath.Base(path), SizeBytes: n, Metadata: []Field{}, Warnings: []Warning{}, ToolVersion: "xolograph-inspector/1",
		Hashes:      Hashes{SHA256: sum("sha256"), SHA512: sum("sha512"), SHA1: sum("sha1"), MD5: sum("md5")},
		InspectedAt: time.Now().UTC(), ParsedBytes: int64(len(head.buf)), Truncated: n > int64(len(head.buf)),
		Limitations: []string{
			"Metadata is written by whatever created or edited the file. It can be missing, stripped, altered or fabricated, and it does not prove who made the file, when, where or with what device.",
			"File type is inferred from content, not from the name. A type match does not mean the file is genuine, unmodified or safe.",
			"Hashes identify exact bytes only. A different hash means different content; an identical hash means identical content, nothing more.",
			"Analysis was done locally; the file was opened read-only and was not executed, rendered or extracted.",
		},
	}
	if info.Size() != n {
		r.warn("caution", "size_changed", "The file's size changed while it was being read; hashes describe the bytes that were read.")
	}
	if r.Truncated {
		r.warn("info", "truncated", fmt.Sprintf("Only the first %d MiB were examined for metadata; hashes cover the whole file.", MaxParseBytes>>20))
	}
	data := head.buf
	r.Type = detectType(data, r.Name)

	switch r.Type.Kind {
	case "jpeg":
		r.safe("JPEG", func() { parseJPEG(r, data) })
	case "png":
		r.safe("PNG", func() { parsePNG(r, data) })
	case "pdf":
		r.safe("PDF", func() { parsePDF(r, data) })
	case "zip":
		r.safe("ZIP", func() { parseZip(r, f, n) })
	case "ole2":
		r.warn("info", "legacy_office", "Legacy Microsoft Office (OLE2) format: document properties are not extracted by this inspector.")
	case "text", "json", "csv", "html", "svg", "xml", "script":
		r.safe("text", func() { parseText(r, data) })
	}
	if r.Image == nil && (r.Type.Kind == "jpeg" || r.Type.Kind == "png" || r.Type.Kind == "gif") {
		r.safe("image header", func() {
			if cfg, format, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
				r.Image = &ImageInfo{Format: format, Width: cfg.Width, Height: cfg.Height}
			} else {
				r.warn("caution", "image_header", "The image header could not be read (the file may be truncated or malformed).")
			}
		})
	}
	if r.Image != nil && int64(r.Image.Width)*int64(r.Image.Height) > maxPixels {
		r.warn("caution", "huge_image", "The declared image dimensions are extremely large; other software may use excessive memory if it opens this file.")
	}

	finishType(r)
	if r.Type.Category == "executable" {
		r.warn("caution", "executable", "This is executable or script content. It was inspected only: it was not run, and it should not be opened outside an isolated environment.")
	}
	if r.SizeBytes == 0 {
		r.warn("info", "empty", "The file is empty.")
	}
	if want != "" {
		act := sum(algo)
		r.Integrity = &Integrity{Algorithm: strings.ToUpper(algo), Expected: want, Actual: act, Match: want == act}
	}
	return r, nil
}

func parseExpected(s string) (hexStr, algo string, err error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return "", "", nil
	}
	if i := strings.Index(s, ":"); i > 0 && i <= 8 {
		pre := strings.Trim(s[:i], "-")
		s = strings.TrimSpace(s[i+1:])
		if pre != "sha256" && pre != "sha1" && pre != "md5" && pre != "sha512" && pre != "sha-256" {
			return "", "", fmt.Errorf("%w: unknown hash prefix", ErrInvalid)
		}
	}
	s = strings.ReplaceAll(s, " ", "")
	if !hexRe.MatchString(s) {
		return "", "", fmt.Errorf("%w: expected hash must be hexadecimal", ErrInvalid)
	}
	switch len(s) {
	case 32:
		algo = "md5"
	case 40:
		algo = "sha1"
	case 64:
		algo = "sha256"
	case 128:
		algo = "sha512"
	default:
		return "", "", fmt.Errorf("%w: expected hash must be an MD5 (32), SHA-1 (40), SHA-256 (64) or SHA-512 (128) hex digest", ErrInvalid)
	}
	return s, algo, nil
}
