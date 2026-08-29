package ticketpdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
)

// compressOutput controls PDF stream compression. Always true outside tests.
var compressOutput = true

// A4 geometry, in millimetres.
const (
	pageWidth  = 210.0
	pageHeight = 297.0
	margin     = 18.0
	contentW   = pageWidth - 2*margin

	// A 60 mm QR prints comfortably larger than the ~20 mm most scanners need,
	// which is what keeps it readable from a phone camera at arm's length.
	qrSizeMM = 60.0
)

// Ticket is everything printed on one admission ticket.
//
// There is deliberately no payment field here: SRS 4.7 requires that card
// details and other unnecessary sensitive data never reach a printed ticket.
type Ticket struct {
	EventTitle   string
	StartsAt     time.Time
	EndsAt       time.Time
	Timezone     string // IANA name, e.g. Asia/Almaty
	VenueName    string
	VenueAddress string

	AttendeeName  string
	AttendeeEmail string

	TicketTypeName string
	TicketCode     string
	TicketID       string
	OrderNumber    string

	// Assigned seating, when the event uses it (SRS 4.3.1).
	SeatSection string
	SeatRow     string
	SeatNumber  string

	// QRToken is the exact string the QR code encodes, always "TKT_<uuid>".
	QRToken string
}

// Render produces a print-ready A4 PDF for one ticket.
//
// Everything is pure black on white: no information is carried by colour alone,
// so a grayscale print loses nothing (SRS 4.7, WCAG 2.1 AA).
func Render(t Ticket) ([]byte, error) {
	if t.QRToken == "" {
		return nil, fmt.Errorf("ticket has no QR token")
	}

	qr, err := QRPNG(t.QRToken)
	if err != nil {
		return nil, err
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	// Compression is on in production. The package's own tests turn it off so
	// they can read the page's text back deterministically: a compressed
	// content stream is binary, and picking text out of it means pattern
	// matching over bytes that change with every random ticket id.
	pdf.SetCompression(compressOutput)
	pdf.SetTitle(pdfSafe("BiletFlow ticket "+t.TicketCode), true)
	pdf.SetAuthor("BiletFlow", true)
	pdf.SetCreator("BiletFlow", true)
	pdf.SetMargins(margin, margin, margin)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	drawHeader(pdf)
	y := drawEvent(pdf, t)
	y = drawDetails(pdf, t, y)
	y = drawQR(pdf, t, qr, y)
	drawIdentifiers(pdf, t, y)
	drawFooter(pdf)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("write pdf: %w", err)
	}
	return buf.Bytes(), nil
}

func drawHeader(pdf *fpdf.Fpdf) {
	pdf.SetFillColor(0, 0, 0)
	pdf.Rect(0, 0, pageWidth, 26, "F")

	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetXY(margin, 7)
	pdf.CellFormat(contentW/2, 12, "BiletFlow", "", 0, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 11)
	pdf.SetXY(margin+contentW/2, 7)
	pdf.CellFormat(contentW/2, 12, "ADMISSION TICKET", "", 0, "R", false, 0, "")

	pdf.SetTextColor(0, 0, 0)
}

func drawEvent(pdf *fpdf.Fpdf, t Ticket) float64 {
	y := 40.0

	label(pdf, margin, y, "EVENT")

	pdf.SetFont("Helvetica", "B", 22)
	pdf.SetXY(margin, y+5)
	// MultiCell wraps a long title onto a second line instead of clipping it.
	pdf.MultiCell(contentW, 9, pdfSafe(truncate(t.EventTitle, 80)), "", "L", false)

	return pdf.GetY() + 6
}

func drawDetails(pdf *fpdf.Fpdf, t Ticket, y float64) float64 {
	col := contentW / 2

	// The time is rendered in the event's own timezone, which SRS 4.11 requires
	// so a ticket read anywhere still shows the time at the venue.
	loc, err := time.LoadLocation(t.Timezone)
	if err != nil {
		loc = time.UTC
	}
	start := t.StartsAt.In(loc)
	end := t.EndsAt.In(loc)

	when := start.Format("Mon 2 Jan 2006, 15:04")
	until := "until " + end.Format("15:04") + " (" + t.Timezone + ")"

	venue := t.VenueName
	if venue == "" {
		venue = "Venue to be announced"
	}

	y = row(pdf, y, col,
		field{"WHEN", when, until},
		field{"WHERE", venue, t.VenueAddress},
	)

	y = row(pdf, y, col,
		field{"ATTENDEE", t.AttendeeName, t.AttendeeEmail},
		field{"TICKET TYPE", t.TicketTypeName, ""},
	)

	// Only printed for an assigned-seating event.
	if t.SeatSection != "" || t.SeatRow != "" || t.SeatNumber != "" {
		seat := fmt.Sprintf("Section %s, Row %s, Seat %s", t.SeatSection, t.SeatRow, t.SeatNumber)
		y = row(pdf, y, col, field{"SEAT", seat, ""}, field{"", "", ""})
	}

	return y
}

// field is one labelled value with an optional second line.
type field struct {
	label     string
	value     string
	secondary string
}

// row draws up to two fields side by side and returns the next y.
func row(pdf *fpdf.Fpdf, y, col float64, left, right field) float64 {
	for i, f := range []field{left, right} {
		if f.label == "" {
			continue
		}
		x := margin + float64(i)*col

		label(pdf, x, y, f.label)

		pdf.SetFont("Helvetica", "B", 13)
		pdf.SetXY(x, y+4.5)
		pdf.CellFormat(col-4, 6, pdfSafe(truncate(f.value, 34)), "", 0, "L", false, 0, "")

		if f.secondary != "" {
			pdf.SetFont("Helvetica", "", 9)
			pdf.SetTextColor(90, 90, 90)
			pdf.SetXY(x, y+11)
			pdf.CellFormat(col-4, 5, pdfSafe(truncate(f.secondary, 48)), "", 0, "L", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
		}
	}
	return y + 22
}

func drawQR(pdf *fpdf.Fpdf, t Ticket, qr []byte, y float64) float64 {
	const imageName = "ticket-qr"

	pdf.RegisterImageOptionsReader(imageName,
		fpdf.ImageOptions{ImageType: "PNG", ReadDpi: false}, bytes.NewReader(qr))

	x := (pageWidth - qrSizeMM) / 2

	// A white plate behind the code guarantees the quiet zone survives even if
	// the page is printed onto tinted stock.
	pdf.SetFillColor(255, 255, 255)
	pdf.Rect(x-4, y-4, qrSizeMM+8, qrSizeMM+8, "F")

	pdf.ImageOptions(imageName, x, y, qrSizeMM, qrSizeMM, false,
		fpdf.ImageOptions{ImageType: "PNG"}, 0, "")

	tokenY := y + qrSizeMM + 5

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(90, 90, 90)
	pdf.SetXY(margin, tokenY)
	pdf.CellFormat(contentW, 4, "Present this code at the entrance", "", 0, "C", false, 0, "")

	// The token in text as well: if a scanner fails, staff can key it in.
	pdf.SetFont("Courier", "", 10)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetXY(margin, tokenY+5)
	pdf.CellFormat(contentW, 5, t.QRToken, "", 0, "C", false, 0, "")

	return tokenY + 16
}

func drawIdentifiers(pdf *fpdf.Fpdf, t Ticket, y float64) {
	pdf.SetDrawColor(200, 200, 200)
	pdf.Line(margin, y, pageWidth-margin, y)
	y += 6

	col := contentW / 2
	row(pdf, y, col,
		field{"TICKET ID", t.TicketCode, t.TicketID},
		field{"ORDER", t.OrderNumber, ""},
	)
}

func drawFooter(pdf *fpdf.Fpdf) {
	pdf.SetDrawColor(200, 200, 200)
	pdf.Line(margin, pageHeight-24, pageWidth-margin, pageHeight-24)

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(110, 110, 110)
	pdf.SetXY(margin, pageHeight-20)
	pdf.MultiCell(contentW, 4,
		"This ticket admits one person once. The QR code is scanned at entry and "+
			"cannot be reused. Issued by BiletFlow as a demonstration; the payment "+
			"behind it was simulated and no money moved.",
		"", "C", false)
	pdf.SetTextColor(0, 0, 0)
}

// label draws a small uppercase caption above a value.
func label(pdf *fpdf.Fpdf, x, y float64, text string) {
	pdf.SetFont("Helvetica", "", 7.5)
	pdf.SetTextColor(120, 120, 120)
	pdf.SetXY(x, y)
	pdf.CellFormat(80, 4, text, "", 0, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
}
