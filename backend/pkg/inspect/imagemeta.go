package inspect

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"regexp"
	"strings"
)

func (r *Report) applyExif(x *exifResult) {
	for _, f := range x.fields {
		r.Metadata = append(r.Metadata, f)
	}
	if x.gps != nil {
		r.GPS = x.gps
		r.add("GPS", "Latitude", fmt.Sprintf("%.6f", x.gps.Latitude))
		r.add("GPS", "Longitude", fmt.Sprintf("%.6f", x.gps.Longitude))
		if x.gps.Altitude != nil {
			r.add("GPS", "Altitude", fmt.Sprintf("%.1f m", *x.gps.Altitude))
		}
		r.warn("sensitive", "gps", "This file contains GPS coordinates. Sharing it could disclose where it was recorded. The coordinates are self-declared and may be wrong or fabricated.")
	}
	if x.serial {
		r.warn("sensitive", "device_serial", "This file contains a camera or lens serial number, which can link it to a specific device.")
	}
	if x.ownerOrOther {
		r.warn("sensitive", "personal_metadata", "This file contains an owner, artist, copyright or unique-ID field that may identify a person or device.")
	}
	if x.thumbnail {
		r.warn("caution", "thumbnail", "An embedded thumbnail is present. Thumbnails are not always updated when an image is edited, so it can show an earlier version.")
	}
	if x.makerNote {
		r.warn("info", "maker_note", "A manufacturer-specific MakerNote block is present and was not decoded.")
	}
}

var xmpFieldRe = regexp.MustCompile(`(?s)(?:xmp:(CreatorTool|CreateDate|ModifyDate|MetadataDate)|photoshop:(DateCreated|City|State|Country))(?:="([^"]{0,200})"|>([^<]{0,200})<)`)

func (r *Report) parseXMP(b []byte) {
	r.add("XMP", "XMP packet", "present")
	n := 0
	for _, m := range xmpFieldRe.FindAllSubmatch(b, 20) {
		name := string(m[1])
		if name == "" {
			name = string(m[2])
		}
		v := string(m[3])
		if v == "" {
			v = string(m[4])
		}
		r.add("XMP", name, v)
		if bytes.Equal(m[2], []byte("City")) || bytes.Equal(m[2], []byte("State")) || bytes.Equal(m[2], []byte("Country")) {
			n++
		}
	}
	if n > 0 {
		r.warn("sensitive", "xmp_location", "The XMP metadata contains place names (city, state or country), which could disclose a location if the file is shared.")
	}
}

// parseJPEG walks the marker segments up to the image data and reads EXIF, XMP, IPTC presence and comments.
func parseJPEG(r *Report, b []byte) {
	i := 2
	segs := 0
	for i+4 <= len(b) && segs < 200 {
		if b[i] != 0xFF {
			r.warn("caution", "jpeg_structure", "The JPEG marker structure is unexpected; metadata may be incomplete.")
			return
		}
		m := b[i+1]
		if m == 0xFF { // fill byte
			i++
			continue
		}
		if m == 0xDA || m == 0xD9 { // start of scan / end of image
			return
		}
		if m == 0x01 || m == 0xD8 || (m >= 0xD0 && m <= 0xD7) { // markers without a length
			i += 2
			continue
		}
		segs++
		l := int(binary.BigEndian.Uint16(b[i+2:]))
		if l < 2 || i+2+l > len(b) {
			r.warn("caution", "jpeg_truncated", "A JPEG segment extends past the end of the file; the file may be truncated or damaged.")
			return
		}
		seg := b[i+4 : i+2+l]
		switch {
		case m == 0xE1 && bytes.HasPrefix(seg, []byte("Exif\x00\x00")):
			x, err := parseTIFFExif(seg[6:])
			if err != nil {
				r.warn("caution", "exif_malformed", "The EXIF block could not be read: "+err.Error()+".")
			} else {
				r.applyExif(x)
			}
		case m == 0xE1 && bytes.HasPrefix(seg, []byte("http://ns.adobe.com/xap/1.0/\x00")):
			r.parseXMP(seg[29:])
		case m == 0xED && bytes.HasPrefix(seg, []byte("Photoshop 3.0")):
			r.add("IPTC", "Photoshop/IPTC block", "present (not decoded)")
			r.warn("caution", "iptc", "A Photoshop/IPTC block is present. It can carry captions, credits and keywords; it was not decoded.")
		case m == 0xFE:
			r.add("JPEG", "Comment", string(seg))
		}
		i += 2 + l
	}
}

// parsePNG reads text chunks, the last-modification time and any embedded EXIF.
func parsePNG(r *Report, b []byte) {
	i := 8
	chunks := 0
	for i+12 <= len(b) && chunks < 10000 {
		chunks++
		l := int(binary.BigEndian.Uint32(b[i:]))
		typ := string(b[i+4 : i+8])
		if l < 0 || i+12+l > len(b) {
			r.warn("caution", "png_truncated", "A PNG chunk extends past the end of the file; the file may be truncated or damaged.")
			return
		}
		data := b[i+8 : i+8+l]
		switch typ {
		case "tEXt":
			if k := bytes.IndexByte(data, 0); k > 0 {
				r.add("PNG text", string(data[:k]), string(data[k+1:]))
				if isPersonalPNGKey(string(data[:k])) {
					r.warn("sensitive", "png_author", "A PNG text field ("+clean(string(data[:k]))+") may identify an author or origin.")
				}
			}
		case "iTXt":
			if k := bytes.IndexByte(data, 0); k > 0 && len(data) > k+3 {
				key := string(data[:k])
				compressed := data[k+1] == 1
				rest := data[k+3:]
				for n := 0; n < 2; n++ { // skip language tag and translated keyword
					if z := bytes.IndexByte(rest, 0); z >= 0 {
						rest = rest[z+1:]
					}
				}
				switch {
				case key == "XML:com.adobe.xmp" && !compressed:
					r.parseXMP(rest)
				case compressed:
					r.add("PNG text", key, "compressed text (not decoded)")
				default:
					r.add("PNG text", key, string(rest))
				}
			}
		case "zTXt":
			if k := bytes.IndexByte(data, 0); k > 0 {
				r.add("PNG text", string(data[:k]), "compressed text (not decoded)")
			}
		case "tIME":
			if len(data) == 7 {
				r.add("PNG", "Last modified (tIME chunk)", fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", binary.BigEndian.Uint16(data), data[2], data[3], data[4], data[5], data[6]))
			}
		case "eXIf":
			if x, err := parseTIFFExif(data); err == nil {
				r.applyExif(x)
			} else {
				r.warn("caution", "exif_malformed", "The EXIF block could not be read: "+err.Error()+".")
			}
		}
		i += 12 + l
	}
}

func isPersonalPNGKey(k string) bool {
	switch strings.ToLower(k) {
	case "author", "artist", "copyright", "creator", "source", "email", "owner":
		return true
	}
	return false
}
