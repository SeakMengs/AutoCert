package main

import (
	"fmt"
	"os"

	"github.com/SeakMengs/AutoCert/pkg/autocert"
)

func main() {
	// Make sure to change input, output as needed
	input := "autocert_tmp/1751286159524726258_signature.svg"
	output := "autocert_tmp/out_bluesign.pdf"
	err := autocert.SvgToPdf(input, output, 219.44, 130.268)
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

	fmt.Printf("PDF Size: Width = %.2f px, Height = %.2f px\n", wPx, hPx)
	fmt.Println("PDF conversion completed successfully. Output file:", output)
}
