package ticketpdf

import (
	_ "embed"

	"github.com/go-pdf/fpdf"
)

// The ticket font.
//
// SRS 7 requires Kazakh and Russian to render on a ticket. The core PDF fonts
// (Helvetica and friends) are cp1252-encoded and have no Cyrillic at all, which
// is why this used to transliterate "Алматы" into "Almaty" - readable, but not
// the attendee's name and not what the SRS asks for.
//
// DejaVu Sans Condensed is embedded instead. It covers Cyrillic including the
// Kazakh-specific letters (ә қ ң ө ұ ү һ і) and, usefully, the tenge sign ₸.
// It ships with the fpdf module this package already depends on, under the
// Bitstream Vera and Arev licences reproduced in fonts/LICENSE-fpdf.txt.
//
// Condensed rather than regular: a ticket is a dense A4 layout with long venue
// names in two columns, and the condensed cut fits more before truncation
// without dropping the point size.
//
//go:embed fonts/DejaVuSansCondensed.ttf
var fontRegular []byte

//go:embed fonts/DejaVuSansCondensed-Bold.ttf
var fontBold []byte

// bodyFont is the family name the renderer asks for. Everything on the ticket
// uses it, so there is no path by which a stray SetFont could fall back to a
// core font and silently lose the Cyrillic again.
const bodyFont = "DejaVu"

// registerFonts embeds the TrueType faces into the document.
//
// AddUTF8FontFromBytes takes the raw TTF, so the font travels inside the
// binary: no font directory to deploy, and no runtime failure on a machine
// that happens not to have DejaVu installed.
func registerFonts(pdf *fpdf.Fpdf) {
	pdf.AddUTF8FontFromBytes(bodyFont, "", fontRegular)
	pdf.AddUTF8FontFromBytes(bodyFont, "B", fontBold)
}
