package autocert

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/noelyahan/impexp"
	"github.com/noelyahan/mergi"
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

func ResizeImage(inFile, outFile string, width, height float64) error {
	img, err := mergi.Import(impexp.NewFileImporter(inFile))
	if err != nil {
		return err
	}

	resized, err := mergi.Resize(img, uint(width), uint(height))
	if err != nil {
		return err
	}

	err = mergi.Export(impexp.NewFileExporter(resized, outFile))
	if err != nil {
		return err
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
		return err
	}
	defer svgData.Close()

	svg, err := canvas.ParseSVG(svgData)
	if err != nil {
		return err
	}

	svg.Fit(pxToMM(0))

	if err := renderers.Write(outFile, svg); err != nil {
		return err
	}

	return ResizePdf(outFile, outFile, []string{"1"}, width, height)
}
