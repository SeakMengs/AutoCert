package autocert

import (
	"fmt"
	"path/filepath"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// Apply pdf or image watermark to a PDF file,
// if array of selected pages is provided, will apply to those pages
// otherwise apply to all pages
func ApplyWatermarkToPdf(inFile string, outFile string, selectedPages []string, watermarkFile string, posX, posY float64) error {
	ext := filepath.Ext(watermarkFile)
	// In pdfcpu, y is inverted
	// For context, in front-end, we calculate position anchor from top-left corner, pos: tl means the anchor is at top-left corner
	// As for scale, it is for image size, 1 means 100% of original size
	// For rotation, it is in degree, default is 45 degree
	description := fmt.Sprintf("pos: tl, off:%.1f %.1f, scale:1 abs, rotation:0", posX, posY*-1)
	onTop := true

	switch ext {
	case ".pdf":
		return api.AddPDFWatermarksFile(inFile, outFile, selectedPages, onTop, watermarkFile, description, nil)
	case ".png", ".jpg", ".jpeg":
		return api.AddImageWatermarksFile(inFile, outFile, selectedPages, onTop, watermarkFile, description, nil)
	default:
		return fmt.Errorf("unsupported watermark file type: %s", ext)
	}
}

// Apply qr code to the bottom right corner of a PDF file
// if array of selected pages is provided, will apply to those pages
// otherwise apply to all pages
func EmbedQRCodeToPdf(inFile, outFile, qrCodePath string, selectePage []string) error {
	description := "pos: br, off: 0 0, scale: 1 abs, rotation: 0"
	err := api.AddImageWatermarksFile(inFile, outFile, selectePage, true, qrCodePath, description, nil)
	if err != nil {
		return fmt.Errorf("failed to embed QR code in PDF: %w", err)
	}
	return nil
}
