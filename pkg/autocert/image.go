package autocert

import (
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/draw"
	"math"
	"os"
	"sync"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/nfnt/resize"
	"github.com/noelyahan/impexp"
	"github.com/noelyahan/mergi"
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
		// Mimics object-contain w-full h-full where w and h are the specified width and height
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

func svgHtml(width, height float64, base64Svg string) string {
	return fmt.Sprintf(`
<html>
<head>
  <style>
	html, body {
	  margin: 0;
	  padding: 0;
	  width: %fin;
	  height: %fin;
	}
	body {
	  display: flex;
	  align-items: center;
	  justify-content: center;
	}
	img {
	  object-fit: contain;
	  width: 100%%;
	  height: 100%%;
	  display: block;
	}
  </style>
</head>
<body>
  <img src="data:image/svg+xml;base64,%s" alt="Signature" />
</body>
</html>`, width, height, base64Svg)
}

func SvgToPdf(inFile, outFile string, width, height float64) error {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	svgBytes, err := os.ReadFile(inFile)
	if err != nil {
		return fmt.Errorf("failed to read SVG: %w", err)
	}

	DPI := 72.0
	// chrome print page use inch
	widthInch := width / DPI
	heightInch := height / DPI

	base64Svg := base64.StdEncoding.EncodeToString(svgBytes)
	if base64Svg == "" {
		return fmt.Errorf("failed to encode SVG to base64")
	}

	html := svgHtml(widthInch, heightInch, base64Svg)

	fmt.Printf("Converting SVG to PDF: %s -> %s (%.2f x %.2f px)\n", inFile, outFile, width, height)

	var pdfBuf []byte
	err = chromedp.Run(ctx,
		// Credit: https://stackoverflow.com/questions/75339208/golang-chromedp-pdf-file-download-without-saving-in-server
		chromedp.Navigate("about:blank"),
		// set the page content and wait until the page is loaded (including its resources).
		chromedp.ActionFunc(func(ctx context.Context) error {
			lctx, cancel := context.WithCancel(ctx)
			defer cancel()
			var wg sync.WaitGroup
			wg.Add(1)
			chromedp.ListenTarget(lctx, func(ev interface{}) {
				if _, ok := ev.(*page.EventLoadEventFired); ok {
					// It's a good habit to remove the event listener if we don't need it anymore.
					cancel()
					wg.Done()
				}
			})

			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}

			if err := page.SetDocumentContent(frameTree.Frame.ID, html).Do(ctx); err != nil {
				return err
			}
			wg.Wait()
			return nil
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfBuf, _, err = page.PrintToPDF().
				WithPreferCSSPageSize(true).
				WithPaperWidth(widthInch).
				WithPaperHeight(heightInch).
				WithMarginTop(0).
				WithMarginBottom(0).
				WithMarginLeft(0).
				WithMarginRight(0).
				WithPrintBackground(true).Do(ctx)
			if err != nil {
				return err
			}
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to render PDF: %w", err)
	}

	if err := os.WriteFile(outFile, pdfBuf, 0644); err != nil {
		return fmt.Errorf("failed to write PDF: %w", err)
	}

	return ResizePDFKeepOrientation(outFile, outFile, []string{"1"}, width, height)
}

// Another way to convert SVG to PDF using tdewolff/canvas but has error when resizing
// func SvgToPdf(inFile, outFile string, width, height float64) error {
// 	if filepath.Ext(inFile) != ".svg" {
// 		return fmt.Errorf("input file is not an SVG: %s", inFile)
// 	}

// 	if filepath.Ext(outFile) != ".pdf" {
// 		return fmt.Errorf("output file is not a PDF: %s", outFile)
// 	}

// 	svgData, err := os.Open(inFile)
// 	if err != nil {
// 		return err
// 	}
// 	defer svgData.Close()

// 	svg, err := canvas.ParseSVG(svgData)
// 	if err != nil {
// 		return err
// 	}

// 	// Don't use fit, in order to keep svg's original size including margins, whitespace
// 	// svg.Fit(pxToMM(0))

// 	if err := renderers.Write(outFile, svg); err != nil {
// 		return err
// 	}

// 	return ResizePDFKeepOrientation(outFile, outFile, []string{"1"}, width, height)
// }
