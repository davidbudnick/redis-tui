package decode

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/klauspost/compress/s2"
	"google.golang.org/protobuf/encoding/protowire"
)

// Result is a successful decode of binary Redis string payload.
type Result struct {
	// Format is a short label such as "protobuf" or "s2+protobuf".
	Format string
	// Text is a schema-less pretty-print of the message (protoc --decode_raw style).
	Text string
	// RawSize is the original Redis value length in bytes.
	RawSize int
	// DecodedSize is the size after decompression (equals RawSize when uncompressed).
	DecodedSize int
}

const (
	// maxDecodeText caps pretty-printed output so huge menus stay usable.
	maxDecodeText = 64 * 1024
	// maxDepth limits nested message expansion.
	maxDepth = 8
	// maxFields caps how many fields are rendered per message level.
	maxFields = 5000
)

// TryBinary attempts s2 decompression then schema-less protobuf decoding.
func TryBinary(raw []byte) (Result, bool) {
	if len(raw) == 0 {
		return Result{}, false
	}

	if decompressed, err := s2.Decode(nil, raw); err == nil && len(decompressed) > 0 {
		if text, ok := formatProtobuf(decompressed); ok {
			return Result{
				Format:      "s2+protobuf",
				Text:        text,
				RawSize:     len(raw),
				DecodedSize: len(decompressed),
			}, true
		}
	}

	if text, ok := formatProtobuf(raw); ok {
		return Result{
			Format:      "protobuf",
			Text:        text,
			RawSize:     len(raw),
			DecodedSize: len(raw),
		}, true
	}

	return Result{}, false
}

// LooksLikeProtobuf reports whether raw is s2-compressed or raw protobuf.
func LooksLikeProtobuf(raw []byte) bool {
	if len(raw) == 0 {
		return false
	}
	if decompressed, err := s2.Decode(nil, raw); err == nil && len(decompressed) > 0 {
		if isValidProtobuf(decompressed) {
			return true
		}
	}
	return isValidProtobuf(raw)
}

// isValidProtobuf walks the full wire payload; every field must parse cleanly.
func isValidProtobuf(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	return consumeMessage(b) == len(b)
}

// consumeMessage returns bytes consumed if b is a valid message, else -1.
func consumeMessage(b []byte) int {
	i := 0
	fields := 0
	for i < len(b) {
		tag, n := protowire.ConsumeVarint(b[i:])
		if n < 0 {
			return -1
		}
		i += n
		num := protowire.Number(tag >> 3)
		wt := protowire.Type(tag & 7)
		if num < 1 {
			return -1
		}
		switch wt {
		case protowire.VarintType:
			_, n = protowire.ConsumeVarint(b[i:])
		case protowire.Fixed64Type:
			_, n = protowire.ConsumeFixed64(b[i:])
		case protowire.BytesType:
			_, n = protowire.ConsumeBytes(b[i:])
		case protowire.Fixed32Type:
			_, n = protowire.ConsumeFixed32(b[i:])
		default:
			// Groups (3/4) and unknown wire types are rejected.
			return -1
		}
		if n < 0 {
			return -1
		}
		i += n
		fields++
	}
	if fields == 0 {
		return -1
	}
	return i
}

// formatProtobuf pretty-prints a wire message; ok is false if invalid.
func formatProtobuf(b []byte) (string, bool) {
	if !isValidProtobuf(b) {
		return "", false
	}
	var sb strings.Builder
	writeMessage(&sb, b, 0)
	return sb.String(), true
}

// writeMessage appends a pretty-printed message (caller validated the wire).
func writeMessage(sb *strings.Builder, b []byte, depth int) {
	i := 0
	fields := 0
	for i < len(b) {
		if sb.Len() >= maxDecodeText {
			fmt.Fprintf(sb, "%s… truncated\n", indent(depth))
			return
		}
		if fields >= maxFields {
			fmt.Fprintf(sb, "%s… more fields\n", indent(depth))
			return
		}
		tag, n := protowire.ConsumeVarint(b[i:])
		i += n
		num := protowire.Number(tag >> 3)
		wt := protowire.Type(tag & 7)

		switch wt {
		case protowire.VarintType:
			v, n := protowire.ConsumeVarint(b[i:])
			i += n
			fmt.Fprintf(sb, "%s%d: %d\n", indent(depth), num, v)
		case protowire.Fixed64Type:
			v, n := protowire.ConsumeFixed64(b[i:])
			i += n
			fmt.Fprintf(sb, "%s%d: 0x%x\n", indent(depth), num, v)
		case protowire.BytesType:
			v, n := protowire.ConsumeBytes(b[i:])
			i += n
			writeBytesField(sb, int(num), v, depth)
		case protowire.Fixed32Type:
			v, n := protowire.ConsumeFixed32(b[i:])
			i += n
			fmt.Fprintf(sb, "%s%d: 0x%x\n", indent(depth), num, v)
		}
		fields++
	}
}

// writeBytesField renders a length-delimited field as string, nested message, or bytes.
func writeBytesField(sb *strings.Builder, num int, v []byte, depth int) {
	if isPrintableUTF8(v) {
		fmt.Fprintf(sb, "%s%d: %q\n", indent(depth), num, string(v))
		return
	}
	if depth < maxDepth && isValidProtobuf(v) {
		fmt.Fprintf(sb, "%s%d: {\n", indent(depth), num)
		writeMessage(sb, v, depth+1)
		fmt.Fprintf(sb, "%s}\n", indent(depth))
		return
	}
	fmt.Fprintf(sb, "%s%d: <%d bytes>\n", indent(depth), num, len(v))
}

// isPrintableUTF8 reports whether b is valid UTF-8 with no control chars (except tab/newline).
func isPrintableUTF8(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	if !utf8.Valid(b) {
		return false
	}
	for _, r := range string(b) {
		if r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		if r < 32 || r == 0x7f {
			return false
		}
	}
	return true
}

// indent returns depth*2 spaces.
func indent(depth int) string {
	if depth <= 0 {
		return ""
	}
	return strings.Repeat("  ", depth)
}
