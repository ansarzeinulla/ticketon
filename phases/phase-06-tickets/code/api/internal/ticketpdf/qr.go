// Package ticketpdf renders admission QR codes and print-ready A4 tickets.
package ticketpdf

import (
	"fmt"

	qrcode "github.com/skip2/go-qrcode"
)

// qrPixels is the pixel size of the generated QR image. Placed at qrSizeMM on
// the page it works out to roughly 430 DPI, well above what any print or screen
// scan needs, so the modules stay crisp rather than blurring into each other.
const qrPixels = 1024

// QRPNG encodes content as a PNG QR code.
//
// High error correction (~30% recoverable) is used deliberately: a ticket gets
// folded, creased and photographed at an angle in a queue, and the payload is
// short enough that the extra redundancy costs nothing.
//
// The image is pure black on pure white with the standard quiet zone, so it
// scans just as well from a grayscale print as from a colour one (SRS 4.7).
func QRPNG(content string) ([]byte, error) {
	if content == "" {
		return nil, fmt.Errorf("qr content must not be empty")
	}

	png, err := qrcode.Encode(content, qrcode.High, qrPixels)
	if err != nil {
		return nil, fmt.Errorf("encode qr code: %w", err)
	}
	return png, nil
}
