package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/SeakMengs/AutoCert/pkg/autocert"
)

func main() {
	const (
		fontDir    = "fonts"
		outputFile = "font_metadata.json"
	)

	fonts, err := autocert.ScanFontDir(fontDir)
	if err != nil {
		log.Fatalf("Failed to scan font directory: %v", err)
	}

	data, err := json.MarshalIndent(fonts, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	// The file can be read by the owner (you), read by users in the file's group, and read by anyone else on the system
	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		log.Fatalf("Failed to write JSON file: %v", err)
	}

	fmt.Printf("Saved metadata for %d fonts to %q\n", len(fonts), outputFile)
}
