package autocert

import (
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/SeakMengs/AutoCert/internal/util"
	"github.com/chai2010/webp"
	"github.com/gen2brain/go-fitz"
	"github.com/nfnt/resize"
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
		EnforceOrient: false,
	}
	err := api.ResizeFile(inFile, outFile, selectedPage, &resizeModel, nil)
	if err != nil {
		return err
	}

	return nil
}

// Extracts a page from a PDF and converts it to PNG without resizing and lose quality.
func PdfToPngByPage(inFile, outDir, selectedPage string) (string, error) {
	img, base := extractPageImage(inFile, outDir, selectedPage)
	if img == nil {
		return "", fmt.Errorf("failed to extract image for page %s", selectedPage)
	}

	outPath := filepath.Join(outDir, base+".png")
	file, err := os.Create(outPath)

	if err != nil {
		return "", err
	}
	defer file.Close()

	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(file, img); err != nil {
		return "", err
	}

	return outPath, nil
}

type ThumbnailFormat int

const (
	ThumbnailFormatWebP ThumbnailFormat = iota
	ThumbnailFormatPNG
)

// Extracts a page, resizes it but maintain aspect ratio, and converts to choosen output format.
func PdfToThumbnailByPage(inFile, outDir, selectedPage string, maxWidth, maxHeight uint, outFormat ThumbnailFormat) (string, error) {
	img, base := extractPageImage(inFile, outDir, selectedPage)
	if img == nil {
		return "", fmt.Errorf("failed to extract image for page %s", selectedPage)
	}

	thumb := resize.Thumbnail(maxWidth, maxHeight, img, resize.Lanczos3)

	var outPath string
	switch outFormat {
	case ThumbnailFormatWebP:
		outPath = filepath.Join(outDir, base+".webp")
		file, err := os.Create(outPath)
		if err != nil {
			return "", err
		}
		defer file.Close()

		if err := webp.Encode(file, thumb, &webp.Options{Lossless: true, Quality: 100}); err != nil {
			return "", err
		}
	case ThumbnailFormatPNG:
		outPath = filepath.Join(outDir, base+".png")
		file, err := os.Create(outPath)
		if err != nil {
			return "", err
		}
		defer file.Close()

		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		if err := encoder.Encode(file, thumb); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported thumbnail format")
	}

	return outPath, nil
}

// Extracts a single page image from PDF.
// return image and base name of the extracted PDF file
func extractPageImage(inFile, outDir, page string) (image.Image, string) {
	extractedPdf, err := ExtractPdfByPage(inFile, outDir, page)
	if err != nil {
		return nil, ""
	}
	defer os.Remove(extractedPdf)

	doc, err := fitz.New(extractedPdf)
	if err != nil {
		return nil, ""
	}
	defer doc.Close()

	img, err := doc.Image(0)
	if err != nil {
		return nil, ""
	}

	base := strings.TrimSuffix(filepath.Base(extractedPdf), filepath.Ext(extractedPdf))
	return img, base
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
	if err := api.OptimizeFile(inFile, outFile, nil); err != nil {
		return err
	}

	return nil
}

// OptimizePdf that accept multipart header and return path to the optimized file
func OptimizePdf(srcFile multipart.FileHeader, outfile string) error {
	src, err := srcFile.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	tmpFile, err := util.CreateTemp("autocert_optimized_*.pdf")
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

// return as width and height in px
func GetPdfSizeByPage(rs io.ReadSeeker, pageNum int) (float64, float64, error) {
	ctx, err := api.ReadAndValidate(rs, model.NewDefaultConfiguration())
	if err != nil {
		return 0, 0, err
	}

	if ctx.PageCount < 1 {
		return 0, 0, fmt.Errorf("pdf has no pages")
	}

	if pageNum < 1 || pageNum > ctx.PageCount {
		return 0, 0, fmt.Errorf("page number %d is out of range (max page count is %d)", pageNum, ctx.PageCount)
	}

	dims, err := ctx.PageDims()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get page dimensions: %v", err)
	}

	if pageNum-1 < 0 || pageNum-1 >= len(dims) {
		return 0, 0, fmt.Errorf("failed to get dimensions for page %d", pageNum)
	}

	dim := dims[pageNum-1]
	return dim.Width, dim.Height, nil
}

func ExtractPdfByPage(inFile string, outDir string, selectedPage string) (string, error) {
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return "", err
		}
	}

	if err := api.ExtractPagesFile(inFile, outDir, []string{selectedPage}, nil); err != nil {
		return "", err
	}

	base := filepath.Base(inFile)
	ext := filepath.Ext(inFile)
	fileName := strings.TrimSuffix(base, ext)
	srcPdf := filepath.Join(outDir, fmt.Sprintf("%s_page_%s%s", fileName, selectedPage, ext))

	return srcPdf, nil
}
