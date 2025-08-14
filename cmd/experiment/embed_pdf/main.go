package main

import (
	"fmt"
	"os"

	"github.com/SeakMengs/AutoCert/pkg/autocert"
)

func main() {
	input := "autocert_tmp/image.svg"
	output := "autocert_tmp/smallw_sign_resized.pdf"
	w, h := 140.0, 90.0
	err := autocert.SvgToPdf(input, output, w, h)
	if err != nil {
		fmt.Println("Error converting SVG to PDF:", err)
	}

	src, err := os.Open(output)
	if err != nil {
		fmt.Println("Error opening PDF file:", err)
		return
	}
	defer src.Close()

	wPx, hPx, err := autocert.GetPdfSizeByPage(src, 1)
	if err != nil {
		fmt.Println("Error getting PDF size:", err)
	}

	fmt.Printf("Expected PDF Size: Width = %.2f px, Height = %.2f px\n", w, h)
	fmt.Printf("PDF Size: Width = %.2f px, Height = %.2f px\n", wPx, hPx)
	fmt.Println("PDF conversion completed successfully. Output file:", output)

	// x, y := 0.0, 514.07996
	x, y := 0.0, 505.50

	pdf := "autocert_tmp/ChongCham template.pdf"
	// pdf := "autocert_tmp/certificate_merged.pdf"
	outpdf := "autocert_tmp/embeded_watermark.pdf"

	srcPdf, err := os.Open(pdf)
	if err != nil {
		fmt.Println("Error opening PDF file:", err)
		return
	}
	defer srcPdf.Close()

	pdfWidth, pdfHeight, err := autocert.GetPdfSizeByPage(srcPdf, 1)
	if err != nil {
		fmt.Println("Error getting PDF size:", err)
		return
	}
	fmt.Printf("PDF to embed Width: %.2f, PDF Height: %.2f\n", pdfWidth, pdfHeight)

	err = autocert.ApplyWatermarkToPdf(pdf, outpdf, []string{"1"}, output, x, y)
	if err != nil {
		fmt.Println("Error applying watermark to PDF:", err)
		return
	}
	fmt.Println("Watermark applied successfully. Output file:", outpdf)
}
