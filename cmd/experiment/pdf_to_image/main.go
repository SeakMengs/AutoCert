package main

import (
	"fmt"

	"github.com/SeakMengs/AutoCert/pkg/autocert"
)

func main() {
	// pdfFilePath := "autocert_tmp/certificate_merged.pdf"
	// pdfFilePath := "autocert_tmp/signature.pdf"
	pdfFilePath := "autocert_tmp/smallw_sign_resized.pdf"
	outputDir := "autocert_tmp/tmp"
	output, err := autocert.PdfToPngByPage(pdfFilePath, outputDir, "1")
	if err != nil {
		panic(err)
	}

	// thumbnail
	// thumbnailOutput, err := autocert.PdfToThumbnailByPage(pdfFilePath, outputDir, "1", 256, 256, autocert.ThumbnailFormatPNG)
	// if err != nil {
	// 	panic(err)
	// }

	fmt.Printf("PDF converted to PNG: %s\n", output)
	// fmt.Printf("PDF converted to thumbnail : %s\n", thumbnailOutput)
}
