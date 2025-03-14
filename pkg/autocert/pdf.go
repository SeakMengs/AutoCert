package autocert

import (
	"fmt"
	"path/filepath"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// Apply pdf or image watermark to a PDF file,
// if array of selected pages is provided, will apply to those pages
// otherwise apply to all pages
func ApplyWatermarkToPdf(inFile string, outFile string, selectedPages []string, watermarkFile string, posX, posY float64) error {
	ext := filepath.Ext(watermarkFile)
	// In pdfcpu, y is inverted
	// For context, in front-end, we calculate position anchor from top-left corner, pos: tl means the anchor is at top-left corner
	// As for scale, it is for image size, 1 means 100% of original size
	// For rotation, it is in degree, default is 45 degree
	description := fmt.Sprintf("pos: tl, off:%.1f %.1f, scale:1 abs, rotation:0", posX, posY*-1)
	onTop := true
	var err error

	switch ext {
	case ".pdf":
		err = api.AddPDFWatermarksFile(inFile, outFile, selectedPages, onTop, watermarkFile, description, nil)
	case ".png", ".jpg", ".jpeg":
		err = api.AddImageWatermarksFile(inFile, outFile, selectedPages, onTop, watermarkFile, description, nil)
	default:
		err = fmt.Errorf("unsupported watermark file type: %s", ext)
	}

	if err != nil {
		return err
	}
	return nil
}

// Apply qr code to the bottom right corner of a PDF file
// if array of selected pages is provided, will apply to those pages
// otherwise apply to all pages
func EmbedQRCodeToPdf(inFile, outFile, qrCodeFile string, selectedPages []string) error {
	ext := filepath.Ext(qrCodeFile)
	description := "pos: br, off: 0 0, scale: 1 abs, rotation: 0"
	onTop := true
	var err error

	switch ext {
	case ".pdf":
		err = api.AddPDFWatermarksFile(inFile, outFile, selectedPages, onTop, qrCodeFile, description, nil)
	case ".png", ".jpg", ".jpeg":
		err = api.AddImageWatermarksFile(inFile, outFile, selectedPages, onTop, qrCodeFile, description, nil)
	default:
		err = fmt.Errorf("unsupported watermark file type: %s", ext)
	}

	if err != nil {
		return err
	}
	return nil
}

func ResizePdf(inFile, outFile string, selectedPage []string, width, height float64) error {
	resizeModel := model.Resize{
		PageDim: &types.Dim{
			Width:  width,
			Height: height,
		},
		UserDim:       true,
		EnforceOrient: true,
	}
	err := api.ResizeFile(inFile, outFile, selectedPage, &resizeModel, nil)
	if err != nil {
		return err
	}

	return nil
}
