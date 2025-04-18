package main

import (
	"fmt"

	"github.com/SeakMengs/AutoCert/pkg/autocert"
)

func main() {
	// pdfFilePath := "autocert_tmp/certificate_merged.pdf"
	pdfFilePath := "autocert_tmp/signature.pdf"
	outputDir := "autocert_tmp/tmp"
	output, err := autocert.PdfToPngByPage(pdfFilePath, outputDir, "1")
	if err != nil {
		panic(err)
	}

	fmt.Printf("PDF to PNG conversion successful. Output file: %s\n", *output)
}
