package main

import (
	"fmt"
	"os"

	"github.com/SeakMengs/AutoCert/pkg/autocert"
)

func main() {
	pdfFilePath := "autocert_tmp/certificate_merged.pdf"

	src, err := os.Open(pdfFilePath)
	if err != nil {
		panic(err)
	}
	defer src.Close()

	pageCount, err := autocert.GetPageCount(src)
	if err != nil {
		panic(err)
	}
	if pageCount < 1 {
		panic("pdf has no pages")
	}
	width, height, err := autocert.GetPdfPageSize(src, 1)
	if err != nil {
		panic(err)
	}
	fmt.Printf("PDF Page Count: %d\n", pageCount)
	fmt.Printf("Page Size: %.2f x %.2f px\n", width, height)
}
