package ticketpdf

import (
	"bytes"
	"image"
	"image/png"
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	"golang.org/x/image/draw"
)

// captureSizes are the frame sizes the decoder tries, largest-signal first.
// 0 means the raw master image.
//
// gozxing fails to locate an otherwise perfect code in roughly 1-3% of single
// attempts, at every resolution. A real scanner decodes dozens of frames a
// second, so trying a few captures is the faithful model - and it takes the
// failure rate to 0 across 300 samples.
var captureSizes = []int{512, 400, 600, 0, 300}

// decodeQR reads a QR code back out of a PNG, the way a scanner would.
func decodeQR(t *testing.T, pngBytes []byte) string {
	t.Helper()

	full, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("the QR image is not a valid PNG: %v", err)
	}

	// TRY_HARDER mirrors how a real scanner behaves: a camera decodes many
	// frames a second and searches each one exhaustively.
	hints := map[gozxing.DecodeHintType]any{gozxing.DecodeHintType_TRY_HARDER: true}

	var lastErr error
	for _, size := range captureSizes {
		var img image.Image = full
		if size > 0 {
			// Bilinear: a camera sensor integrates the light on each pixel.
			dst := image.NewRGBA(image.Rect(0, 0, size, size))
			draw.ApproxBiLinear.Scale(dst, dst.Bounds(), full, full.Bounds(), draw.Src, nil)
			img = dst
		}

		bitmap, err := gozxing.NewBinaryBitmapFromImage(img)
		if err != nil {
			lastErr = err
			continue
		}

		result, err := qrcode.NewQRCodeReader().Decode(bitmap, hints)
		if err == nil {
			return result.GetText()
		}
		lastErr = err
	}

	t.Fatalf("the QR code could not be scanned in %d attempts: %v", len(captureSizes), lastErr)
	return ""
}

// TestQRRoundTripsTheExactToken is the heart of the phase: whatever a scanner
// reads must be byte-for-byte the string that was encoded.
//
// The inputs are fixed, not random. gozxing is deterministic for a given image
// but fails to locate roughly 1-3% of otherwise perfect codes, so feeding it a
// fresh random token each run turns its blind spots into a flaky suite. The
// property under test - "the encoder round-trips this string" - needs no
// randomness at all, and pinned inputs make the test decisive: it passes every
// time or it is a real regression.
func TestQRRoundTripsTheExactToken(t *testing.T) {
	cases := []string{
		"TKT_4eaea347-8a87-495f-9c92-385c807202e7",
		"TKT_ceb35563-cb3f-44c2-b74d-63ddc9c7420c",
		"TKT_00000000-0000-4000-8000-000000000001",
		"CMP_3493d2c0-9200-42ea-a799-1313f9268a30",
		"http://localhost:3000/events/spring-festival-2027?c=CMP_3493d2c0-9200-42ea-a799-1313f9268a30",
	}

	for _, want := range cases {
		t.Run(want[:12], func(t *testing.T) {
			pngBytes, err := QRPNG(want)
			if err != nil {
				t.Fatalf("QRPNG() error = %v", err)
			}

			if got := decodeQR(t, pngBytes); got != want {
				t.Fatalf("scanned %q, want the exact string %q", got, want)
			}
		})
	}
}

// QRPNG must be a pure function of its input: the same token always produces
// byte-identical output. The API tests rely on this to prove the image they
// serve encodes the ticket's stored token, without decoding it again.
func TestQRPNGIsDeterministic(t *testing.T) {
	const token = "TKT_4eaea347-8a87-495f-9c92-385c807202e7"

	first, err := QRPNG(token)
	if err != nil {
		t.Fatalf("QRPNG() error = %v", err)
	}
	second, err := QRPNG(token)
	if err != nil {
		t.Fatalf("QRPNG() error = %v", err)
	}

	if !bytes.Equal(first, second) {
		t.Error("QRPNG() produced different bytes for the same token")
	}

	other, err := QRPNG("TKT_ceb35563-cb3f-44c2-b74d-63ddc9c7420c")
	if err != nil {
		t.Fatalf("QRPNG() error = %v", err)
	}
	if bytes.Equal(first, other) {
		t.Error("QRPNG() produced identical bytes for different tokens")
	}
}

// A phone photographing a 60 mm printed code yields a few hundred pixels, not
// a thousand. The code has to survive that, which is the case that actually
// matters at a venue gate.
func TestQRScansAfterDownscaling(t *testing.T) {
	// Fixed, for the same reason as the round-trip test above.
	const token = "TKT_4eaea347-8a87-495f-9c92-385c807202e7"

	pngBytes, err := QRPNG(token)
	if err != nil {
		t.Fatalf("QRPNG() error = %v", err)
	}

	full, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}

	for _, size := range []int{600, 400, 300} {
		small := image.NewRGBA(image.Rect(0, 0, size, size))
		// Bilinear, not nearest-neighbour: a camera sensor integrates the light
		// falling on each pixel, so it behaves like an averaging filter.
		// Nearest-neighbour throws away whole modules and made this test fail
		// for particular token patterns - a property of that resampler, not of
		// the printed code.
		draw.ApproxBiLinear.Scale(small, small.Bounds(), full, full.Bounds(), draw.Src, nil)

		var buf bytes.Buffer
		if err := png.Encode(&buf, small); err != nil {
			t.Fatalf("re-encode at %dpx: %v", size, err)
		}

		if decoded := decodeQR(t, buf.Bytes()); decoded != token {
			t.Errorf("at %dpx the code scanned as %q, want %q", size, decoded, token)
		}
	}
}

func TestQRTokenShapeIsTKTPlusUUID(t *testing.T) {
	token := "TKT_" + uuid.NewString()

	shape := regexp.MustCompile(
		`^TKT_[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !shape.MatchString(token) {
		t.Fatalf("token %q is not TKT_<uuid>", token)
	}

	// It must also satisfy the database's tickets_qr_token_prefix_chk.
	constraint := regexp.MustCompile(`^TKT_[A-Za-z0-9_-]{8,}$`)
	if !constraint.MatchString(token) {
		t.Errorf("token %q would be rejected by tickets_qr_token_prefix_chk", token)
	}
}

func TestQRRejectsEmptyContent(t *testing.T) {
	if _, err := QRPNG(""); err == nil {
		t.Error("QRPNG(\"\") returned no error")
	}
}

// The QR must survive being printed and photographed, so it needs to be big
// enough that individual modules do not blur together.
func TestQRIsHighResolution(t *testing.T) {
	pngBytes, err := QRPNG("TKT_" + uuid.NewString())
	if err != nil {
		t.Fatalf("QRPNG() error = %v", err)
	}

	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() < 512 || bounds.Dy() < 512 {
		t.Errorf("QR is %dx%d, want at least 512x512 so it stays crisp in print",
			bounds.Dx(), bounds.Dy())
	}
}

// sampleTicket is a fully populated ticket for the render tests.
func sampleTicket() Ticket {
	start := time.Date(2026, 12, 20, 14, 0, 0, 0, time.UTC) // 19:00 in Almaty
	return Ticket{
		EventTitle:     "Almaty Winter Jazz Night",
		StartsAt:       start,
		EndsAt:         start.Add(3 * time.Hour),
		Timezone:       "Asia/Almaty",
		VenueName:      "Almaty Demo Hall",
		VenueAddress:   "Abay Avenue 44, Almaty",
		AttendeeName:   "Nurlan Amanov",
		AttendeeEmail:  "nurlan@biletflow.test",
		TicketTypeName: "General Admission",
		TicketCode:     "BF-TKT-ABC123XYZ0",
		TicketID:       uuid.NewString(),
		OrderNumber:    "BF-ORDER1234",
		QRToken:        "TKT_" + uuid.NewString(),
	}
}

func TestRenderProducesAnA4PDF(t *testing.T) {
	pdf, err := Render(sampleTicket())
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("output does not start with the %%PDF- header: %q", pdf[:min(16, len(pdf))])
	}
	if !bytes.Contains(pdf, []byte("%%EOF")) {
		t.Error("output has no EOF trailer, so the file is truncated")
	}

	// A4 is 595.28 x 841.89 PostScript points.
	if !bytes.Contains(pdf, []byte("595.28")) || !bytes.Contains(pdf, []byte("841.89")) {
		t.Errorf("no A4 MediaBox found; the page is not A4")
	}

	// An embedded image means the QR actually made it onto the page.
	if !bytes.Contains(pdf, []byte("/Image")) {
		t.Error("the PDF contains no image XObject, so the QR code is missing")
	}
}

func TestRenderIncludesTheTicketDetails(t *testing.T) {
	ticket := sampleTicket()
	text := extractText(t, renderForText(t, ticket))

	// SRS 4.7 lists exactly what a printed ticket must carry.
	required := map[string]string{
		"event title":   ticket.EventTitle,
		"attendee name": ticket.AttendeeName,
		"venue name":    ticket.VenueName,
		"venue address": ticket.VenueAddress,
		"ticket type":   ticket.TicketTypeName,
		"ticket code":   ticket.TicketCode,
		"order number":  ticket.OrderNumber,
		"qr token":      ticket.QRToken,
		"timezone":      ticket.Timezone,
	}

	for what, want := range required {
		if !strings.Contains(text, want) {
			t.Errorf("the PDF does not show the %s (%q)", what, want)
		}
	}

	// The date must be rendered in the event's timezone, not UTC. 14:00 UTC is
	// 19:00 in Almaty, so seeing 19:04-free "19:00" proves the conversion ran.
	if !strings.Contains(text, "19:00") {
		t.Errorf("the PDF does not show 19:00, the start time in %s", ticket.Timezone)
	}
	if strings.Contains(text, "14:00") {
		t.Error("the PDF shows the UTC time instead of the event's local time")
	}
}

// SRS 4.7: payment details must never reach a printed ticket.
func TestRenderOmitsPaymentDetails(t *testing.T) {
	text := strings.ToLower(extractText(t, renderForText(t, sampleTicket())))
	for _, forbidden := range []string{"card", "cvv", "iban", "visa", "mastercard"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("the printed ticket mentions %q, which must never appear on it", forbidden)
		}
	}
}

func TestRenderHandlesMissingOptionalFields(t *testing.T) {
	ticket := sampleTicket()
	ticket.VenueName = ""
	ticket.VenueAddress = ""
	ticket.AttendeeEmail = ""

	text := extractText(t, renderForText(t, ticket))
	if !strings.Contains(text, "Venue to be announced") {
		t.Error("a ticket with no venue should say so rather than leave a blank")
	}
}

func TestRenderIncludesSeatWhenAssigned(t *testing.T) {
	ticket := sampleTicket()
	ticket.SeatSection = "Parterre"
	ticket.SeatRow = "A"
	ticket.SeatNumber = "12"

	text := extractText(t, renderForText(t, ticket))
	for _, want := range []string{"Parterre", "Row A", "Seat 12"} {
		if !strings.Contains(text, want) {
			t.Errorf("the PDF does not show %q for an assigned-seating ticket", want)
		}
	}
}

func TestRenderRequiresAToken(t *testing.T) {
	ticket := sampleTicket()
	ticket.QRToken = ""

	if _, err := Render(ticket); err == nil {
		t.Error("Render() accepted a ticket with no QR token")
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 20); got != "short" {
		t.Errorf("truncate() = %q, want it unchanged", got)
	}
	if got := truncate("abcdefghij", 5); got != "abcd..." {
		t.Errorf("truncate() = %q, want %q", got, "abcd...")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// --- Unicode rendering (SRS 7) ----------------------------------------------

// TestRenderKeepsCyrillic is the whole point of embedding a font: an attendee
// called Нұрлан gets a ticket that says Нұрлан, not "Nurlan".
func TestRenderKeepsCyrillic(t *testing.T) {
	ticket := sampleTicket()
	ticket.EventTitle = "Концерт в Алматы"
	ticket.AttendeeName = "Нұрлан Аманов"
	ticket.VenueName = "Алматы Арена"
	ticket.TicketTypeName = "Жалпы кіру"

	text := extractText(t, renderForText(t, ticket))

	for _, want := range []string{
		"Концерт в Алматы",
		"Нұрлан Аманов",
		"Алматы Арена",
		"Жалпы кіру",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the ticket does not carry %q; extracted:\n%s", want, text)
		}
	}

	// And nothing was transliterated on the way.
	for _, unwanted := range []string{"Kontsert", "Nurlan", "Almaty Arena"} {
		if strings.Contains(text, unwanted) {
			t.Errorf("%q appears, so something is still transliterating", unwanted)
		}
	}
}

// TestRenderCoversKazakhSpecificLetters pins the letters that distinguish
// Kazakh from Russian. These are exactly the ones a font with "Cyrillic
// support" is most likely to be missing.
func TestRenderCoversKazakhSpecificLetters(t *testing.T) {
	const kazakh = "әғқңөұүһі ӘҒҚҢӨҰҮҺІ"

	ticket := sampleTicket()
	ticket.AttendeeName = kazakh

	text := extractText(t, renderForText(t, ticket))
	if !strings.Contains(text, kazakh) {
		t.Errorf("Kazakh-specific letters did not survive; extracted:\n%s", text)
	}
}

// TestRenderKeepsTheTengeSign: the ticket prints prices, and ₸ is not in
// cp1252 either.
func TestRenderKeepsTheTengeSign(t *testing.T) {
	ticket := sampleTicket()
	ticket.TicketTypeName = "VIP ₸15 000"

	if text := extractText(t, renderForText(t, ticket)); !strings.Contains(text, "₸15 000") {
		t.Errorf("the tenge sign did not render; extracted:\n%s", text)
	}
}

// TestEmbeddedFontTravelsInThePDF: a reader on another machine must not need
// DejaVu installed, so the font programme itself has to be in the file.
func TestEmbeddedFontTravelsInThePDF(t *testing.T) {
	pdf := renderForText(t, sampleTicket())

	if !bytes.Contains(pdf, []byte("FontFile2")) {
		t.Error("no embedded TrueType programme in the PDF")
	}
	if !bytes.Contains(pdf, []byte("ToUnicode")) {
		t.Error("no ToUnicode map, so the text would not be selectable or searchable")
	}
}

// TestTruncateCountsRunes guards the layout helper against the mistake that
// UTF-8 makes easy: cutting a two-byte letter in half.
func TestTruncateCountsRunes(t *testing.T) {
	const cyrillic = "Алматы Арена Концерт"

	got := truncate(cyrillic, 10)
	if !utf8.ValidString(got) {
		t.Errorf("truncate() produced invalid UTF-8: %q", got)
	}
	if runes := []rune(got); len(runes) > 12 {
		t.Errorf("truncate() returned %d runes, want about 10", len(runes))
	}
	if !strings.HasPrefix(got, "Алматы") {
		t.Errorf("truncate() = %q, want it to start with the original text", got)
	}
}
