package autocert

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"os"
	"path/filepath"

	"github.com/nfnt/resize"
	"github.com/noelyahan/impexp"
	"github.com/noelyahan/mergi"
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

/*
* objectContain resizes the image to fit within the given width and height
* while preserving aspect ratio then centers it on a canvas of the specified size.
* key difference is that the output image will always be the specified width and height
* unlike normal preserve aspect ratio resizing which make the output image width and height
* based on the calculated aspect ratio.
* Example: if the input image is 100x50 and the specified width and height are 200x200,
* the output image will be resized to 100x50, then centered on a 200x200 canvas,
* resulting in a final image of 200x200 with the original image centered.
 */
// can remove mergi dependency by writing our own import/export functions
func ResizeImage(inFile, outFile string, width, height float64, objectContain bool) error {
	img, err := mergi.Import(impexp.NewFileImporter(inFile))
	if err != nil {
		return err
	}

	if img == nil {
		return fmt.Errorf("failed to import image: %s", inFile)
	}

	var resized image.Image
	if objectContain {
		origBounds := img.Bounds()
		origWidth := float64(origBounds.Dx())
		origHeight := float64(origBounds.Dy())

		ratioW := width / origWidth
		ratioH := height / origHeight

		ratio := math.Min(ratioW, ratioH)

		newWidth := uint(origWidth * ratio)
		newHeight := uint(origHeight * ratio)

		resizedImg := resize.Resize(newWidth, newHeight, img, resize.Lanczos3)

		canvas := image.NewRGBA(image.Rect(0, 0, int(width), int(height)))

		// Calculate position to center the resized image on the canvas
		// width and height is basically parent container size
		offsetX := (int(width) - int(newWidth)) / 2
		offsetY := (int(height) - int(newHeight)) / 2

		draw.Draw(canvas, image.Rect(offsetX, offsetY, offsetX+int(newWidth), offsetY+int(newHeight)), resizedImg, image.Point{}, draw.Src)

		resized = canvas
	} else {
		resized = resize.Resize(uint(width), uint(height), img, resize.Lanczos3)
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

	// Don't use fit, in order to keep svg's original size including margins, whitespace
	// svg.Fit(pxToMM(0))

	if err := renderers.Write(outFile, svg); err != nil {
		return err
	}

	return ResizePdf(outFile, outFile, []string{"1"}, width, height)
}
