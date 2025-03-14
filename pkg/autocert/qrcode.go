package autocert

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/skip2/go-qrcode"
	qrsvg "github.com/wamuir/svg-qr-code"
)

// If generate qr code for pdf file, size 50 should be enough
func GenerateQRCode(link, outFile string, size int) error {
	err := qrcode.WriteFile(link, qrcode.Highest, size, outFile)
	if err != nil {
		return fmt.Errorf("failed to generate QR code: %w", err)
	}
	return nil
}

func GenerateQRCodeAsPdf(link, outFile string, size int) error {
	if filepath.Ext(outFile) != ".pdf" {
		return fmt.Errorf("output file is not a PDF: %s", outFile)
	}

	qr, err := qrsvg.New(link)
	if err != nil {
		panic(err)
	}

	tmpQrSvg, err := os.CreateTemp("", "autocert_qr_*.svg")
	if err != nil {
		return fmt.Errorf("failed to create temporary SVG file: %w", err)
	}
	defer os.Remove(tmpQrSvg.Name())

	if err := os.WriteFile(tmpQrSvg.Name(), []byte(qr.String()), 0644); err != nil {
		return fmt.Errorf("failed to save SVG QR code to file: %w", err)
	}

	return SvgToPdf(tmpQrSvg.Name(), outFile, float64(size), float64(size))
}
