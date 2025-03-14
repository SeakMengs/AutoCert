package autocert

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/h2non/bimg"
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

// Require libvips to be installed on the system.
func ResizeImage(inFile, outFile string, width, height float64) error {
	buffer, err := bimg.Read(inFile)
	if err != nil {
		return fmt.Errorf("failed to read image: %v", err)
	}

	options := bimg.Options{
		Width:        int(width),
		Height:       int(height),
		Quality:      100,
		Lossless:     true,
		Compression:  0,
		Interpolator: bimg.Bicubic,
		Rotate:       0,
	}

	newImage, err := bimg.NewImage(buffer).Process(options)
	if err != nil {
		return fmt.Errorf("failed to process image: %v", err)
	}

	err = bimg.Write(outFile, newImage)
	if err != nil {
		return fmt.Errorf("failed to write image: %v", err)
	}

	return nil
}

func SvgToPdf(inFile, outFile string, width, height float64) error {
	if filepath.Ext(inFile) != ".svg" {
		return fmt.Errorf("input file is not an SVG: %s", inFile)
	}

	if filepath.Ext(outFile) != ".pdf" {
		return fmt.Errorf("output file is not a PDF: %s", outFile)
	}

	svgData, err := os.Open(inFile)
	if err != nil {
		return fmt.Errorf("failed to read SVG file: %v", err)
	}
	defer svgData.Close()

	svg, err := canvas.ParseSVG(svgData)
	if err != nil {
		return fmt.Errorf("failed to parse SVG: %v", err)
	}

	svg.Fit(pxToMM(0))

	if err := renderers.Write(outFile, svg); err != nil {
		return fmt.Errorf("failed to write PDF: %v", err)
	}

	return ResizePdf(outFile, outFile, []string{"1"}, width, height)
}
