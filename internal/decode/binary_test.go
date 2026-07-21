package decode

import (
	"strings"
	"testing"

	"github.com/klauspost/compress/s2"
	"google.golang.org/protobuf/encoding/protowire"
)

// encodeStringField builds a single length-delimited string field.
func encodeStringField(num protowire.Number, s string) []byte {
	var b []byte
	b = protowire.AppendTag(b, num, protowire.BytesType)
	b = protowire.AppendString(b, s)
	return b
}

// encodeVarintField builds a single varint field.
func encodeVarintField(num protowire.Number, v uint64) []byte {
	var b []byte
	b = protowire.AppendTag(b, num, protowire.VarintType)
	b = protowire.AppendVarint(b, v)
	return b
}

// encodeNested builds field num wrapping an inner message.
func encodeNested(num protowire.Number, inner []byte) []byte {
	var b []byte
	b = protowire.AppendTag(b, num, protowire.BytesType)
	b = protowire.AppendBytes(b, inner)
	return b
}

func TestTryBinary_RawProtobuf(t *testing.T) {
	raw := encodeStringField(1, "rst-demo:DELIVERY")
	raw = append(raw, encodeStringField(3, "America/Toronto")...)
	raw = append(raw, encodeVarintField(5, 42)...)

	got, ok := TryBinary(raw)
	if !ok {
		t.Fatal("TryBinary returned false for valid protobuf")
	}
	if got.Format != "protobuf" {
		t.Errorf("Format = %q, want protobuf", got.Format)
	}
	if got.RawSize != len(raw) {
		t.Errorf("RawSize = %d, want %d", got.RawSize, len(raw))
	}
	if !containsAll(got.Text, []string{`1: "rst-demo:DELIVERY"`, `3: "America/Toronto"`, `5: 42`}) {
		t.Errorf("Text missing expected fields:\n%s", got.Text)
	}
}

func TestTryBinary_S2CompressedProtobuf(t *testing.T) {
	raw := encodeStringField(1, "menu-id")
	raw = append(raw, encodeNested(4, encodeStringField(1, "cat-1"))...)
	compressed := s2.Encode(nil, raw)

	got, ok := TryBinary(compressed)
	if !ok {
		t.Fatal("TryBinary returned false for s2+protobuf")
	}
	if got.Format != "s2+protobuf" {
		t.Errorf("Format = %q, want s2+protobuf", got.Format)
	}
	if got.RawSize != len(compressed) {
		t.Errorf("RawSize = %d, want %d", got.RawSize, len(compressed))
	}
	if got.DecodedSize != len(raw) {
		t.Errorf("DecodedSize = %d, want %d", got.DecodedSize, len(raw))
	}
	if !containsAll(got.Text, []string{`1: "menu-id"`, `4: {`, `1: "cat-1"`}) {
		t.Errorf("Text missing expected nested fields:\n%s", got.Text)
	}
}

func TestTryBinary_RejectsRandomBinary(t *testing.T) {
	raw := make([]byte, 32)
	raw[0] = 0xC1
	raw[10] = 0x01
	if _, ok := TryBinary(raw); ok {
		t.Fatal("TryBinary should reject non-protobuf binary")
	}
}

func TestTryBinary_RejectsEmpty(t *testing.T) {
	if _, ok := TryBinary(nil); ok {
		t.Fatal("empty should fail")
	}
	if _, ok := TryBinary([]byte{}); ok {
		t.Fatal("empty slice should fail")
	}
}

func TestLooksLikeProtobuf(t *testing.T) {
	if LooksLikeProtobuf(nil) || LooksLikeProtobuf([]byte{}) {
		t.Error("empty should not look like protobuf")
	}
	raw := encodeStringField(1, "hello")
	if !LooksLikeProtobuf(raw) {
		t.Error("expected raw protobuf to look like protobuf")
	}
	if !LooksLikeProtobuf(s2.Encode(nil, raw)) {
		t.Error("expected s2 protobuf to look like protobuf")
	}
	if LooksLikeProtobuf([]byte{0xff, 0xfe, 0x00}) {
		t.Error("random binary should not look like protobuf")
	}
	// s2 of non-protobuf
	if LooksLikeProtobuf(s2.Encode(nil, []byte{0xff, 0xfe, 0xfd, 0x00, 0x01})) {
		t.Error("s2 of random bytes should not look like protobuf")
	}
}

func TestIsValidProtobuf_Empty(t *testing.T) {
	if isValidProtobuf(nil) || isValidProtobuf([]byte{}) {
		t.Error("empty invalid")
	}
}

func TestConsumeMessage_Errors(t *testing.T) {
	if consumeMessage([]byte{}) >= 0 {
		t.Error("empty should fail")
	}
	if consumeMessage([]byte{0x00}) >= 0 {
		t.Error("field 0 should fail")
	}
	if consumeMessage([]byte{0x08}) >= 0 {
		t.Error("truncated varint should fail")
	}
	if consumeMessage([]byte{0x09, 0x01}) >= 0 {
		t.Error("truncated fixed64 should fail")
	}
	if consumeMessage([]byte{0x0d, 0x01}) >= 0 {
		t.Error("truncated fixed32 should fail")
	}
	if consumeMessage([]byte{0x0a, 0x05, 0x01}) >= 0 {
		t.Error("truncated bytes should fail")
	}
	if consumeMessage([]byte{0x0b}) >= 0 {
		t.Error("start group should fail")
	}
	if consumeMessage([]byte{0x0f}) >= 0 {
		t.Error("illegal wire type should fail")
	}
	// incomplete leading tag varint
	if consumeMessage([]byte{0x80}) >= 0 {
		t.Error("truncated tag should fail")
	}
}

func TestSplitTag(t *testing.T) {
	num, wt, ok := splitTag(0x08) // field 1, varint
	if !ok || num != 1 || wt != wireVarint {
		t.Errorf("splitTag(0x08) = %d %d %v", num, wt, ok)
	}
	if _, _, ok := splitTag(0); ok {
		t.Error("field 0 should fail")
	}
	// field number past max via high bits
	if _, _, ok := splitTag((maxFieldNumber + 1) << 3); ok {
		t.Error("oversized field should fail")
	}
}

func TestWriteMessage_InvalidTagStops(t *testing.T) {
	// After isValidProtobuf passed, writeMessage trusts tags; call it with
	// a crafted buffer that has a valid first field then a field-0 tag so
	// splitTag fails mid-message (defensive early return).
	var b []byte
	b = append(b, encodeVarintField(1, 1)...)
	b = append(b, 0x00) // field 0 tag
	var sb strings.Builder
	writeMessage(&sb, b, 0)
	if !strings.Contains(sb.String(), "1: 1") {
		t.Errorf("expected first field rendered, got %q", sb.String())
	}
}

func TestFormatProtobuf_FixedTypesAndOpaque(t *testing.T) {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.Fixed64Type)
	b = protowire.AppendFixed64(b, 0x1122334455667788)
	b = protowire.AppendTag(b, 2, protowire.Fixed32Type)
	b = protowire.AppendFixed32(b, 0xaabbccdd)
	b = protowire.AppendTag(b, 3, protowire.BytesType)
	b = protowire.AppendBytes(b, []byte{0x00, 0x01, 0xff})

	text, ok := formatProtobuf(b)
	if !ok {
		t.Fatal("formatProtobuf failed")
	}
	if !strings.Contains(text, "1: 0x") || !strings.Contains(text, "2: 0x") || !strings.Contains(text, "3: <3 bytes>") {
		t.Errorf("unexpected text:\n%s", text)
	}
}

func TestFormatProtobuf_Invalid(t *testing.T) {
	if _, ok := formatProtobuf([]byte{0xff}); ok {
		t.Error("invalid should fail")
	}
}

func TestWriteMessage_TruncationBySize(t *testing.T) {
	var b []byte
	payload := strings.Repeat("x", 200)
	for i := 0; i < 400; i++ {
		b = append(b, encodeStringField(1, payload)...)
	}
	text, ok := formatProtobuf(b)
	if !ok {
		t.Fatal("formatProtobuf failed on large message")
	}
	if !strings.Contains(text, "truncated") {
		t.Errorf("expected truncation marker, got len=%d", len(text))
	}
}

func TestWriteMessage_TruncationByFieldCount(t *testing.T) {
	// Temporarily lower maxFields via many tiny fields — maxFields is 5000.
	// Building 5001 varint fields is fine.
	var b []byte
	for i := 0; i < maxFields+10; i++ {
		b = append(b, encodeVarintField(1, uint64(i%128))...)
	}
	text, ok := formatProtobuf(b)
	if !ok {
		t.Fatal("formatProtobuf failed")
	}
	if !strings.Contains(text, "more fields") {
		t.Errorf("expected field-cap marker:\n%s", text[len(text)-80:])
	}
}

func TestWriteBytesField_MaxDepth(t *testing.T) {
	// Nest messages deeper than maxDepth → innermost becomes opaque bytes.
	inner := encodeVarintField(1, 1)
	for d := 0; d < maxDepth+2; d++ {
		inner = encodeNested(1, inner)
	}
	text, ok := formatProtobuf(inner)
	if !ok {
		t.Fatal("formatProtobuf failed for deep nest")
	}
	if !strings.Contains(text, "bytes>") {
		t.Errorf("expected depth-capped opaque bytes:\n%s", text)
	}
}

func TestIsPrintableUTF8(t *testing.T) {
	if !isPrintableUTF8([]byte("hello")) {
		t.Error("ascii should be printable")
	}
	if !isPrintableUTF8([]byte("ok\t\n\r")) {
		t.Error("tab/newline/return should be allowed")
	}
	if isPrintableUTF8([]byte{0x00, 0x01}) {
		t.Error("control bytes should not be printable")
	}
	if isPrintableUTF8([]byte{}) {
		t.Error("empty should not be printable")
	}
	if isPrintableUTF8([]byte{0x01}) {
		t.Error("SOH control should fail")
	}
	if isPrintableUTF8([]byte{0x7f}) {
		t.Error("DEL should fail")
	}
	if isPrintableUTF8([]byte{0xff, 0xfe}) {
		t.Error("invalid utf8 should fail")
	}
}

func TestIndent(t *testing.T) {
	if indent(0) != "" {
		t.Error("depth 0")
	}
	if indent(2) != "    " {
		t.Errorf("depth 2 = %q", indent(2))
	}
}

func TestTryBinary_EmptyAfterS2(t *testing.T) {
	compressed := s2.Encode(nil, []byte{})
	if _, ok := TryBinary(compressed); ok {
		t.Error("empty decompressed should not succeed")
	}
}

func containsAll(s string, parts []string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
