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

	x, y := 0.0, 514.07996

	pdf := "autocert_tmp/ChongCham template.pdf"
	outpdf := "autocert_tmp/embeded_watermark.pdf"
	err = autocert.ApplyWatermarkToPdf(pdf, outpdf, []string{"1"}, output, x, y)
	if err != nil {
		fmt.Println("Error applying watermark to PDF:", err)
		return
	}
	fmt.Println("Watermark applied successfully. Output file:", outpdf)
}
