package inspect

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

// A deliberately small EXIF/TIFF reader: bounds-checked, depth- and count-limited, and restricted to a curated tag list.

const (
	maxIFDEntries = 512
	maxExifFields = 300
)

type tiffReader struct {
	b     []byte
	order binary.ByteOrder
	seen  map[uint32]bool
	count int
}

func (t *tiffReader) u16(off uint32) (uint16, bool) {
	if uint64(off)+2 > uint64(len(t.b)) {
		return 0, false
	}
	return t.order.Uint16(t.b[off:]), true
}

func (t *tiffReader) u32(off uint32) (uint32, bool) {
	if uint64(off)+4 > uint64(len(t.b)) {
		return 0, false
	}
	return t.order.Uint32(t.b[off:]), true
}

type entry struct {
	tag, typ uint16
	count    uint32
	valOff   uint32 // offset of the value bytes within b
	ok       bool
}

var typeSize = map[uint16]uint32{1: 1, 2: 1, 3: 2, 4: 4, 5: 8, 6: 1, 7: 1, 8: 2, 9: 4, 10: 8, 11: 4, 12: 8}

func (t *tiffReader) entryAt(off uint32) entry {
	e := entry{}
	var ok1, ok2, ok3 bool
	e.tag, ok1 = t.u16(off)
	e.typ, ok2 = t.u16(off + 2)
	e.count, ok3 = t.u32(off + 4)
	if !ok1 || !ok2 || !ok3 {
		return e
	}
	sz, known := typeSize[e.typ]
	if !known {
		return e
	}
	total := uint64(sz) * uint64(e.count)
	if total > uint64(len(t.b)) {
		return e
	}
	if total <= 4 {
		if uint64(off)+8+total > uint64(len(t.b)) {
			return e
		}
		e.valOff = off + 8
	} else {
		v, ok := t.u32(off + 8)
		if !ok || uint64(v)+total > uint64(len(t.b)) {
			return e
		}
		e.valOff = v
	}
	e.ok = true
	return e
}

func (t *tiffReader) str(e entry) string {
	if !e.ok || e.typ != 2 {
		return ""
	}
	s := string(t.b[e.valOff : e.valOff+e.count])
	if i := strings.IndexByte(s, 0); i >= 0 {
		s = s[:i]
	}
	return s
}

func (t *tiffReader) rational(e entry, i uint32) (float64, bool) {
	if !e.ok || (e.typ != 5 && e.typ != 10) || i >= e.count {
		return 0, false
	}
	off := e.valOff + i*8
	n, ok1 := t.u32(off)
	d, ok2 := t.u32(off + 4)
	if !ok1 || !ok2 || d == 0 {
		return 0, false
	}
	if e.typ == 10 {
		return float64(int32(n)) / float64(int32(d)), true
	}
	return float64(n) / float64(d), true
}

func (t *tiffReader) integer(e entry) (uint32, bool) {
	if !e.ok || e.count < 1 {
		return 0, false
	}
	switch e.typ {
	case 3:
		v, ok := t.u16(e.valOff)
		return uint32(v), ok
	case 4:
		return t.u32(e.valOff)
	case 1:
		return uint32(t.b[e.valOff]), true
	}
	return 0, false
}

var ifd0Tags = map[uint16]string{0x010E: "Image description", 0x010F: "Camera make", 0x0110: "Camera model", 0x0131: "Software", 0x0132: "Modified (file)", 0x013B: "Artist", 0x8298: "Copyright"}
var exifTags = map[uint16]string{
	0x9003: "Date/time original", 0x9004: "Date/time digitized", 0x9010: "Offset time", 0x9011: "Offset time (original)", 0xA420: "Image unique ID",
	0xA431: "Body serial number", 0xA430: "Camera owner name", 0xA433: "Lens make", 0xA434: "Lens model", 0xA435: "Lens serial number",
}
var orientation = []string{"", "Normal", "Mirrored", "Rotated 180°", "Mirrored vertically", "Mirrored, rotated 90° CCW", "Rotated 90° CW", "Mirrored, rotated 90° CW", "Rotated 90° CCW"}

type exifResult struct {
	fields       []Field
	gps          *GPS
	thumbnail    bool
	makerNote    bool
	serial       bool
	ownerOrOther bool // owner/artist/unique-id style personal identifiers
}

// parseTIFFExif reads a TIFF structure (as embedded in JPEG APP1 "Exif" or a PNG eXIf chunk).
func parseTIFFExif(b []byte) (*exifResult, error) {
	if len(b) < 8 {
		return nil, fmt.Errorf("EXIF block too short")
	}
	t := &tiffReader{b: b, seen: map[uint32]bool{}}
	switch string(b[:2]) {
	case "II":
		t.order = binary.LittleEndian
	case "MM":
		t.order = binary.BigEndian
	default:
		return nil, fmt.Errorf("EXIF byte order marker missing")
	}
	if m, _ := t.u16(2); m != 42 {
		return nil, fmt.Errorf("not a TIFF structure")
	}
	res := &exifResult{}
	ifd0, _ := t.u32(4)
	next := t.readIFD(res, ifd0, "ifd0", 0)
	if next != 0 { // IFD1 normally describes an embedded thumbnail
		t.readIFD(res, next, "ifd1", 0)
	}
	return res, nil
}

// readIFD walks one directory and returns the offset of the next IFD (0 if none).
func (t *tiffReader) readIFD(res *exifResult, off uint32, kind string, depth int) uint32 {
	if depth > 3 || t.seen[off] || off == 0 {
		return 0
	}
	t.seen[off] = true
	n, ok := t.u16(off)
	if !ok {
		return 0
	}
	if n > maxIFDEntries {
		n = maxIFDEntries
	}
	var exifPtr, gpsPtr uint32
	gpsE := map[uint16]entry{}
	add := func(group, name, val string) {
		if val = clean(val); val != "" && t.count < maxExifFields {
			t.count++
			res.fields = append(res.fields, Field{group, name, val})
		}
	}
	for i := uint32(0); i < uint32(n); i++ {
		e := t.entryAt(off + 2 + i*12)
		if !e.ok {
			continue
		}
		switch kind {
		case "ifd0":
			switch e.tag {
			case 0x8769:
				exifPtr, _ = t.integer(e)
			case 0x8825:
				gpsPtr, _ = t.integer(e)
			case 0x0112:
				if v, ok := t.integer(e); ok && int(v) > 0 && int(v) < len(orientation) {
					add("EXIF", "Orientation", orientation[v])
				}
			default:
				if name, ok := ifd0Tags[e.tag]; ok {
					add("EXIF", name, t.str(e))
					if e.tag == 0x013B || e.tag == 0x8298 {
						res.ownerOrOther = true
					}
				}
			}
		case "exif":
			switch e.tag {
			case 0x829A:
				if v, ok := t.rational(e, 0); ok && v > 0 {
					if v < 1 {
						add("EXIF", "Exposure time", fmt.Sprintf("1/%.0f s", math.Round(1/v)))
					} else {
						add("EXIF", "Exposure time", fmt.Sprintf("%.1f s", v))
					}
				}
			case 0x829D:
				if v, ok := t.rational(e, 0); ok {
					add("EXIF", "Aperture", fmt.Sprintf("f/%.1f", v))
				}
			case 0x8827:
				if v, ok := t.integer(e); ok {
					add("EXIF", "ISO", fmt.Sprint(v))
				}
			case 0x920A:
				if v, ok := t.rational(e, 0); ok {
					add("EXIF", "Focal length", fmt.Sprintf("%.1f mm", v))
				}
			case 0x927C:
				res.makerNote = true
			case 0x9286:
				if e.typ == 7 && e.count > 8 {
					add("EXIF", "User comment", string(t.b[e.valOff+8:e.valOff+e.count]))
				}
			default:
				if name, ok := exifTags[e.tag]; ok {
					add("EXIF", name, t.str(e))
					switch e.tag {
					case 0xA431, 0xA435:
						res.serial = true
					case 0xA430, 0xA420:
						res.ownerOrOther = true
					}
				}
			}
		case "gps":
			gpsE[e.tag] = e
		case "ifd1":
			if e.tag == 0x0201 || e.tag == 0x0202 {
				res.thumbnail = true
			}
		}
	}
	if kind == "gps" {
		t.finishGPS(res, gpsE, add)
	}
	if kind == "ifd0" {
		if exifPtr != 0 {
			t.readIFD(res, exifPtr, "exif", depth+1)
		}
		if gpsPtr != 0 {
			t.readIFD(res, gpsPtr, "gps", depth+1)
		}
	}
	nxt, _ := t.u32(off + 2 + uint32(n)*12)
	return nxt
}

func (t *tiffReader) dms(e entry) (float64, bool) {
	d, ok1 := t.rational(e, 0)
	m, ok2 := t.rational(e, 1)
	s, ok3 := t.rational(e, 2)
	if !ok1 || !ok2 || !ok3 || e.count < 3 {
		return 0, false
	}
	return d + m/60 + s/3600, true
}

func (t *tiffReader) finishGPS(res *exifResult, g map[uint16]entry, add func(group, name, val string)) {
	latE, okLat := g[2]
	lonE, okLon := g[4]
	if okLat && okLon {
		lat, ok1 := t.dms(latE)
		lon, ok2 := t.dms(lonE)
		if ok1 && ok2 {
			if strings.HasPrefix(strings.ToUpper(t.str(g[1])), "S") {
				lat = -lat
			}
			if strings.HasPrefix(strings.ToUpper(t.str(g[3])), "W") {
				lon = -lon
			}
			if math.Abs(lat) <= 90 && math.Abs(lon) <= 180 && !(lat == 0 && lon == 0) {
				res.gps = &GPS{Latitude: math.Round(lat*1e6) / 1e6, Longitude: math.Round(lon*1e6) / 1e6}
				if a, ok := t.rational(g[6], 0); ok {
					if ref, ok := t.integer(g[5]); ok && ref == 1 {
						a = -a
					}
					a = math.Round(a*10) / 10
					res.gps.Altitude = &a
				}
			}
		}
	}
	if ds := t.str(g[0x1D]); ds != "" {
		add("GPS", "GPS date", ds)
	}
	if e, ok := g[7]; ok {
		h, ok1 := t.rational(e, 0)
		m, ok2 := t.rational(e, 1)
		s, ok3 := t.rational(e, 2)
		if ok1 && ok2 && ok3 {
			add("GPS", "GPS time (UTC)", fmt.Sprintf("%02.0f:%02.0f:%02.0f", h, m, s))
		}
	}
}
