package autocert

import (
	"fmt"

	"github.com/skip2/go-qrcode"
)

// If generate qr code for pdf file, size 50 should be enough
func GenerateQRCode(link, outputPath string, size int) error {
	err := qrcode.WriteFile(link, qrcode.Medium, size, outputPath)
	if err != nil {
		return fmt.Errorf("failed to generate QR code: %w", err)
	}
	return nil
}
