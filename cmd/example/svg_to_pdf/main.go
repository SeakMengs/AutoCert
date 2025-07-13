package main

import (
	"fmt"
	"os"

	"github.com/SeakMengs/AutoCert/pkg/autocert"
)

func main() {
	input := "autocert_tmp/smallw_sign.svg"
	output := "autocert_tmp/smallw_sign_resized.pdf"
	w, h := 59.98, 593.1
	// w, h := 132.983, 593.1
	// w, h := 50.983, 30.1
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
}
