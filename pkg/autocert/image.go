package autocert

import (
	"fmt"

	"github.com/h2non/bimg"
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

// tdewolff/canvas uses mm as the unit of measurement
// func SvgToPdf(inFile, outFile string, width, height float64) error {
// 	c := canvas.New(pxToMM(width), pxToMM(height))
// 	canvasCtx := canvas.NewContext(c)
// 	canvasCtx.SetCoordSystem(canvas.CartesianIV)

// 	svgData, err := os.Open(inFile)
// 	if err != nil {
// 		return fmt.Errorf("failed to read SVG file: %v", err)
// 	}

// 	svg, err := canvas.ParseSVG(svgData)
// 	if err != nil {
// 		return fmt.Errorf("failed to parse SVG: %v", err)
// 	}

// 	svg.RenderTo(canvasCtx)
// 	if err := renderers.Write(outFile, c); err != nil {
// 		return fmt.Errorf("failed to write PDF: %v", err)
// 	}
// 	return nil
// }
