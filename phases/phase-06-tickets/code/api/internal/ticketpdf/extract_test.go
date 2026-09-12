package ticketpdf

import (
	"bytes"
	"compress/zlib"
	"io"
	"regexp"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"
)

// PDF content streams sit between these markers.
var streamPattern = regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\nendstream`)

// Text is drawn with the Tj operator, its argument a parenthesised string.
var showTextPattern = regexp.MustCompile(`\(((?:[^()\\]|\\.)*)\)\s*Tj`)

// renderForText renders a ticket with stream compression off, so its text can
// be read back reliably.
//
// Structure - A4 page size, the embedded QR image, the %PDF header - is still
// asserted against the real compressed output elsewhere. Only the text
// assertions use this, because scanning for text inside a compressed stream
// means pattern matching over binary that changes with every random ticket id,
// which made the suite fail roughly one run in ten.
func renderForText(t *testing.T, ticket Ticket) []byte {
	t.Helper()

	compressOutput = false
	t.Cleanup(func() { compressOutput = true })

	pdf, err := Render(ticket)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	return pdf
}

// extractText pulls the visible text out of a rendered PDF by reading the
// arguments of its text-showing operators, exactly as a viewer would.
func extractText(t *testing.T, pdf []byte) string {
	t.Helper()

	var out strings.Builder

	for _, match := range streamPattern.FindAllSubmatch(pdf, -1) {
		content := match[1]

		if inflated, err := inflate(content); err == nil {
			content = inflated
		}

		for _, text := range showTextPattern.FindAllSubmatch(content, -1) {
			out.WriteString(decodeShownText(unescapePDFString(string(text[1]))))
			out.WriteString("\n")
		}
	}

	if out.Len() == 0 {
		t.Fatal("no text could be extracted from the PDF")
	}
	return out.String()
}

func inflate(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	return io.ReadAll(reader)
}

// unescapePDFString reverses the backslash escaping PDF string literals use.
func unescapePDFString(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			i++
			switch s[i] {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			default:
				b.WriteByte(s[i])
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// decodeShownText turns one Tj operand back into readable text.
//
// With an embedded UTF-8 font, fpdf writes strings as UTF-16BE rather than as
// plain bytes - which is precisely what lets a ticket carry Cyrillic. The
// extractor has to undo that, or every assertion in this package would be
// reading interleaved NUL bytes.
//
// A string that is not valid UTF-16BE is returned unchanged, so the helper
// still works on anything written with a core font.
func decodeShownText(raw string) string {
	if len(raw) == 0 || len(raw)%2 != 0 {
		return raw
	}

	units := make([]uint16, 0, len(raw)/2)
	for i := 0; i < len(raw); i += 2 {
		units = append(units, uint16(raw[i])<<8|uint16(raw[i+1]))
	}

	decoded := string(utf16.Decode(units))
	if !utf8.ValidString(decoded) {
		return raw
	}
	return decoded
}
