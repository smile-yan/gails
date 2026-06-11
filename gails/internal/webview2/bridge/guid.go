//go:build windows

// Port of upstream
// github.com/wailsapp/wails/webview2/pkg/edge/guid.go (the parser
// and Stringer; the upstream file was adapted from
// https://github.com/go-ole/go-ole and retains its MIT license
// notice).
//
// Upstream divergence: upstream exposes `NewGUID(string) *GUID` that
// returns nil on parse failure. The Gails port exposes
// `GUIDFromString(string) (*GUID, error)` so call sites can use a
// single error-handling idiom; nil is replaced with a descriptive
// error. The parsing algorithm (32/36/38-char shape dispatch plus
// decodeHexUint32/16/Byte64/Byte/Char) and the `String` formatter
// are ported verbatim.
package bridge

import "fmt"

// hextable is the uppercase hex alphabet used by String() to format
// each nibble of the GUID. Matches upstream.
const hextable = "0123456789ABCDEF"

// emptyGUID is the canonical zero-GUID string, returned by String()
// when the receiver is nil. Matches upstream.
const emptyGUID = "{00000000-0000-0000-0000-000000000000}"

// GUID is the Microsoft COM 128-bit identifier. Its shape matches
// Windows COM IID/CLSID layout so values can be cast interchangeably
// to and from `windows.GUID` and used in raw COM calls.
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

// String formats the GUID as `{XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}`
// in uppercase. A nil receiver returns the empty GUID string,
// matching upstream.
func (g *GUID) String() string {
	if g == nil {
		return emptyGUID
	}

	var c [38]byte
	c[0] = '{'
	putUint32Hex(c[1:9], g.Data1)
	c[9] = '-'
	putUint16Hex(c[10:14], g.Data2)
	c[14] = '-'
	putUint16Hex(c[15:19], g.Data3)
	c[19] = '-'
	putByteHex(c[20:24], g.Data4[0:2])
	c[24] = '-'
	putByteHex(c[25:37], g.Data4[2:8])
	c[37] = '}'
	return string(c[:])
}

// GUIDFromString parses a GUID in one of three accepted shapes:
//
//	XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX         (32 hex chars)
//	XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX      (36 chars with dashes)
//	{XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX}    (38 chars with braces)
//
// Parsing is case-insensitive. A return of (nil, error) indicates a
// malformed input; this differs from upstream's `NewGUID` which
// returns a bare nil for the same condition.
func GUIDFromString(s string) (*GUID, error) {
	d := []byte(s)
	var d1, d2, d3, d4a, d4b []byte

	switch len(d) {
	case 38:
		if d[0] != '{' || d[37] != '}' {
			return nil, fmt.Errorf("GUIDFromString: missing braces in %q", s)
		}
		d = d[1:37]
		fallthrough
	case 36:
		if d[8] != '-' || d[13] != '-' || d[18] != '-' || d[23] != '-' {
			return nil, fmt.Errorf("GUIDFromString: missing dashes in %q", s)
		}
		d1 = d[0:8]
		d2 = d[9:13]
		d3 = d[14:18]
		d4a = d[19:23]
		d4b = d[24:36]
	case 32:
		d1 = d[0:8]
		d2 = d[8:12]
		d3 = d[12:16]
		d4a = d[16:20]
		d4b = d[20:32]
	default:
		return nil, fmt.Errorf("GUIDFromString: %q has length %d, want 32, 36, or 38", s, len(d))
	}

	var g GUID
	var ok1, ok2, ok3, ok4 bool
	g.Data1, ok1 = decodeHexUint32(d1)
	g.Data2, ok2 = decodeHexUint16(d2)
	g.Data3, ok3 = decodeHexUint16(d3)
	g.Data4, ok4 = decodeHexByte64(d4a, d4b)
	if ok1 && ok2 && ok3 && ok4 {
		return &g, nil
	}
	return nil, fmt.Errorf("GUIDFromString: invalid hex digits in %q", s)
}

func decodeHexUint32(src []byte) (value uint32, ok bool) {
	var b1, b2, b3, b4 byte
	var ok1, ok2, ok3, ok4 bool
	b1, ok1 = decodeHexByte(src[0], src[1])
	b2, ok2 = decodeHexByte(src[2], src[3])
	b3, ok3 = decodeHexByte(src[4], src[5])
	b4, ok4 = decodeHexByte(src[6], src[7])
	value = (uint32(b1) << 24) | (uint32(b2) << 16) | (uint32(b3) << 8) | uint32(b4)
	ok = ok1 && ok2 && ok3 && ok4
	return
}

func decodeHexUint16(src []byte) (value uint16, ok bool) {
	var b1, b2 byte
	var ok1, ok2 bool
	b1, ok1 = decodeHexByte(src[0], src[1])
	b2, ok2 = decodeHexByte(src[2], src[3])
	value = (uint16(b1) << 8) | uint16(b2)
	ok = ok1 && ok2
	return
}

func decodeHexByte64(s1 []byte, s2 []byte) (value [8]byte, ok bool) {
	var ok1, ok2, ok3, ok4, ok5, ok6, ok7, ok8 bool
	value[0], ok1 = decodeHexByte(s1[0], s1[1])
	value[1], ok2 = decodeHexByte(s1[2], s1[3])
	value[2], ok3 = decodeHexByte(s2[0], s2[1])
	value[3], ok4 = decodeHexByte(s2[2], s2[3])
	value[4], ok5 = decodeHexByte(s2[4], s2[5])
	value[5], ok6 = decodeHexByte(s2[6], s2[7])
	value[6], ok7 = decodeHexByte(s2[8], s2[9])
	value[7], ok8 = decodeHexByte(s2[10], s2[11])
	ok = ok1 && ok2 && ok3 && ok4 && ok5 && ok6 && ok7 && ok8
	return
}

func decodeHexByte(c1, c2 byte) (value byte, ok bool) {
	var n1, n2 byte
	var ok1, ok2 bool
	n1, ok1 = decodeHexChar(c1)
	n2, ok2 = decodeHexChar(c2)
	value = (n1 << 4) | n2
	ok = ok1 && ok2
	return
}

func decodeHexChar(c byte) (byte, bool) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}

	return 0, false
}

func putUint32Hex(b []byte, v uint32) {
	b[0] = hextable[byte(v>>24)>>4]
	b[1] = hextable[byte(v>>24)&0x0f]
	b[2] = hextable[byte(v>>16)>>4]
	b[3] = hextable[byte(v>>16)&0x0f]
	b[4] = hextable[byte(v>>8)>>4]
	b[5] = hextable[byte(v>>8)&0x0f]
	b[6] = hextable[byte(v)>>4]
	b[7] = hextable[byte(v)&0x0f]
}

func putUint16Hex(b []byte, v uint16) {
	b[0] = hextable[byte(v>>8)>>4]
	b[1] = hextable[byte(v>>8)&0x0f]
	b[2] = hextable[byte(v)>>4]
	b[3] = hextable[byte(v)&0x0f]
}

func putByteHex(dst, src []byte) {
	for i := 0; i < len(src); i++ {
		dst[i*2] = hextable[src[i]>>4]
		dst[i*2+1] = hextable[src[i]&0x0f]
	}
}
