package autocert

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/sunshineplan/imgconv"
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

// Extracts a specific page from a PDF file and converts it to PNG.
// It takes the input PDF file path, output directory path, and the page number to extract.
// If page does not exist, will throw an error.
// Return file path to the converted png image
func PdfToPngByPage(inFile, outDir string, selectedPages string) (*string, error) {
	// Create output directory if it doesn't exist
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return nil, err
		}
	}

	// Extract the selected page from the PDF
	if err := api.ExtractPagesFile(inFile, outDir, []string{selectedPages}, nil); err != nil {
		return nil, err
	}

	// Build the path to the extracted PDF page
	// pdfcpu names output files as: inFile_page_selectedPages.pdf
	base := filepath.Base(inFile)
	ext := filepath.Ext(inFile)
	fileName := strings.TrimSuffix(base, ext)
	srcPdf := filepath.Join(outDir, fmt.Sprintf("%s_page_%s%s", fileName, selectedPages, ext))

	// Clean up the extracted PDF file when we're done
	defer os.Remove(srcPdf)

	// Open the extracted PDF page for conversion
	src, err := imgconv.Open(srcPdf)
	if err != nil {
		return nil, err
	}

	// Create output PNG file
	imgExt := ".png"
	outFilePath := filepath.Join(outDir, fmt.Sprintf("%s_page_%s%s", fileName, selectedPages, imgExt))
	outFile, err := os.Create(outFilePath)
	if err != nil {
		return nil, err
	}
	defer outFile.Close()

	// Convert PDF page to PNG
	if err := imgconv.Write(outFile, src, &imgconv.FormatOption{Format: imgconv.PNG}); err != nil {
		return nil, err
	}

	return &outFilePath, nil
}

// Merge inFiles by concatenation in the order specified and write the result to outfile.
// outfile will be overwritten if it exists.
// Perfectly match for this project, if don't want overwrite, use MergeAppendFile instead.
func MergePdf(inFiles []string, outFile string) error {
	// If divider page is true, a blank page will be inserted between each input file.
	dividerPage := false
	return api.MergeCreateFile(inFiles, outFile, dividerPage, nil)
}

// Optimize pdf will also validate the pdf itself
func OptimizePdfFile(inFile, outFile string) error {
	// Optimize the PDF context
	if err := api.OptimizeFile(inFile, outFile, nil); err != nil {
		return err
	}

	return nil
}

// OptimizePdf that accept multipart header and return path to the optimized file
func OptimizePdf(srcFile multipart.FileHeader, outfile string) error {
	// Open the uploaded file
	src, err := srcFile.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	tmpFile, err := os.CreateTemp("", "optimized_*.pdf")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, src); err != nil {
		return err
	}
	tmpFile.Close()

	if err := OptimizePdfFile(tmpFile.Name(), tmpFile.Name()); err != nil {
		return err
	}

	// Move the optimized file to the specified output path
	if err := os.Rename(tmpFile.Name(), outfile); err != nil {
		return err
	}

	return nil
}

func GetPageCount(rs io.ReadSeeker) (int, error) {
	ctx, err := api.ReadAndValidate(rs, model.NewDefaultConfiguration())
	if err != nil {
		return 0, err
	}

	return ctx.PageCount, nil
}

func ExtractPdfByPage(inFile string, outDir string, selectedPages string) (string, error) {
	// Create output directory if it doesn't exist
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return "", err
		}
	}

	// Extract the selected page from the PDF
	if err := api.ExtractPagesFile(inFile, outDir, []string{selectedPages}, nil); err != nil {
		return "", err
	}

	// Build the path to the extracted PDF page
	base := filepath.Base(inFile)
	ext := filepath.Ext(inFile)
	fileName := strings.TrimSuffix(base, ext)
	srcPdf := filepath.Join(outDir, fmt.Sprintf("%s_page_%s%s", fileName, selectedPages, ext))

	return srcPdf, nil
}
