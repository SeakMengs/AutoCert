package autocert

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/skip2/go-qrcode"
	qrsvg "github.com/wamuir/svg-qr-code"
)

// If generate qr code for pdf file, size 50 should be enough
func GenerateQRCode(link, outFile string, size int) error {
	err := qrcode.WriteFile(link, qrcode.Highest, size, outFile)
	if err != nil {
		return err
	}
	return nil
}

func GenerateQRCodeAsPdf(link, outFile string, size int) error {
	if filepath.Ext(outFile) != ".pdf" {
		return fmt.Errorf("output file is not a PDF: %s", outFile)
	}

	qr, err := qrsvg.New(link)
	if err != nil {
		return err
	}

	tmpQrSvg, err := util.CreateTemp("autocert_qr_*.svg")
	if err != nil {
		return err
	}
	defer os.Remove(tmpQrSvg.Name())

	if err := os.WriteFile(tmpQrSvg.Name(), []byte(qr.String()), 0644); err != nil {
		return err
	}

	return SvgToPdf(tmpQrSvg.Name(), outFile, float64(size), float64(size))
}

// Generate qr based on pdf page's dimension, the qr code size will 6% of the page width
func GenerateQRCodeAsPdfByPdfPage(link, pdfFile string, pageNum int, outFile string) error {
	if filepath.Ext(outFile) != ".pdf" {
		return fmt.Errorf("output file is not a PDF: %s", outFile)
	}

	pdfSrc, err := os.Open(pdfFile)
	if err != nil {
		return fmt.Errorf("failed to open PDF file: %w", err)
	}
	defer pdfSrc.Close()

	w, h, err := GetPdfSizeByPage(pdfSrc, pageNum)
	if err != nil {
		return fmt.Errorf("failed to get PDF page size: %w", err)
	}
	if w <= 0 || h <= 0 {
		return fmt.Errorf("invalid PDF page size: width=%.2f, height=%.2f", w, h)
	}
	size := int(w * 0.06)
	maxSize := 200
	minSize := 50

	if size > maxSize {
		size = maxSize
	}

	if size < minSize {
		size = minSize
	}

	return GenerateQRCodeAsPdf(link, outFile, size)
}
