package main

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/SeakMengs/AutoCert/pkg/autocert"
)

// AnnotationItem represents a single annotation from the JSON
type AnnotationItem struct {
	ID            string  `json:"id"`
	Type          string  `json:"type"`
	X             float64 `json:"x"`
	Y             float64 `json:"y"`
	Value         string  `json:"value,omitempty"`
	Width         float64 `json:"width"`
	Height        float64 `json:"height"`
	FontName      string  `json:"fontName,omitempty"`
	FontSize      int     `json:"fontSize,omitempty"`
	FontWeight    string  `json:"fontWeight,omitempty"`
	FontColor     string  `json:"fontColor,omitempty"`
	Color         string  `json:"color,omitempty"`
	SignatureData string  `json:"signatureData,omitempty"`
	Email         string  `json:"email,omitempty"`
	Status        string  `json:"status,omitempty"`
}

// AnnotationsJSON represents the structure of the JSON annotations
type AnnotationsJSON map[string][]AnnotationItem

func main() {
	now := time.Now()

	cfg := autocert.NewDefaultConfig()

	// Define paths for the template PDF and CSV.
	templatePath := "autocert_tmp/certificate_merged.pdf"
	// tempSignaturePath := "autocert_tmp/svg_signature.pdf"
	// tempSignaturePath := "autocert_tmp/svg_signature_pdfcpu.pdf"
	tempSignaturePath := "autocert_tmp/signature.svg"
	csvPath := "autocert_tmp/example.csv"
	font := "Microsoft YaHei"

	// Sample JSON data from the paste.txt file
	jsonData := `{
  "1": [
    {
      "id": "-TFoKkQcOIxXMACN9g8iB",
      "type": "signature",
      "x": 702.14,
      "y": 0,
      "width": 140,
      "height": 90,
      "signatureData": "signatureData",
      "email": "lol@gmail.com",
      "status": "not_invited",
      "color": "#FFC4C4"
    },
    {
      "id": "oQdk4uWOWw70ZsVnPq78J",
      "type": "column",
      "x": 181.0302501548872,
      "y": 294.1944304774588,
      "value": "name3",
      "width": 478.05481796023753,
      "height": 40,
      "fontName": "Arial",
      "fontSize": 24,
      "fontWeight": "regular",
      "fontColor": "#000000",
      "color": "#FFC4C4"
    }
  ],
  "2": [
    {
      "id": "Dc3G0GXWw0XXFcysyWPdK",
      "type": "column",
      "x": 411,
      "y": 408,
      "value": "name3",
      "width": 251,
      "height": 40,
      "fontName": "Arial",
      "fontSize": 24,
      "fontWeight": "regular",
      "fontColor": "#000000",
      "color": "#FFC4C4"
    }
  ]
}`

	// Parse the JSON data
	var annotations AnnotationsJSON
	if err := json.Unmarshal([]byte(jsonData), &annotations); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	// Create PageAnnotations
	pageAnnotations := autocert.PageAnnotations{
		PageSignatureAnnotations: make(map[uint][]autocert.SignatureAnnotate),
		PageColumnAnnotations:    make(map[uint][]autocert.ColumnAnnotate),
	}

	// Process each page's annotations
	for pageStr, items := range annotations {
		var page uint
		// Convert the page string to an integer
		fmt.Sscanf(pageStr, "%d", &page)

		for _, item := range items {
			switch item.Type {
			case "column":
				columnAnnotate := autocert.ColumnAnnotate{
					BaseAnnotate: autocert.BaseAnnotate{
						ID:   item.ID,
						Type: autocert.AnnotateTypeColumn,
						Position: autocert.Position{
							X: item.X,
							Y: item.Y,
						},
						Size: autocert.Size{
							Width:  item.Width,
							Height: item.Height,
						},
					},
					Value:      item.Value,
					FontName:   font,
					FontColor:  item.FontColor,
					FontSize:   float64(item.FontSize),
					FontWeight: autocert.FontWeightRegular,
				}
				pageAnnotations.PageColumnAnnotations[page] = append(
					pageAnnotations.PageColumnAnnotations[page],
					columnAnnotate,
				)

			case "signature":
				signatureAnnotate := autocert.SignatureAnnotate{
					BaseAnnotate: autocert.BaseAnnotate{
						ID:   item.ID,
						Type: autocert.AnnotateTypeSignature,
						Position: autocert.Position{
							X: item.X,
							Y: item.Y,
						},
						Size: autocert.Size{
							Width:  item.Width,
							Height: item.Height,
						},
					},
					SignatureFilePath: tempSignaturePath,
					Email:             item.Email,
				}
				pageAnnotations.PageSignatureAnnotations[page] = append(
					pageAnnotations.PageSignatureAnnotations[page],
					signatureAnnotate,
				)
			}
		}
	}

	// Define rendering settings.
	settings := autocert.Settings{
		TextFitRectBox:       true,
		RemoveLineBreaksBool: true,
		EmbedQRCode:          true,
	}

	// Create a CertificateGenerator.
	generator := autocert.NewCertificateGenerator("lol", templatePath, csvPath, cfg, pageAnnotations, settings)

	// Generate certificates.
	// The outputFilePattern is a format string; here, certificates will be named "certificate_0.pdf", "certificate_1.pdf", etc.
	generatedFiles, err := generator.Generate("certificate_%d.pdf")
	if err != nil {
		log.Fatalf("Certificate generation failed: %v", err)
	}

	fmt.Println("Generated certificate files:")
	for _, file := range generatedFiles {
		absPath, _ := filepath.Abs(file)
		fmt.Println(absPath)
	}

	then := time.Now()
	fmt.Printf("Time taken: %v for %d certificate \n", then.Sub(now), len(generatedFiles))
	fmt.Println("All done!")
}
