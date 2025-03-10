package autocert

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdewolff/canvas"
	"golang.org/x/image/font/sfnt"
)

type FontWeight string

const (
	FontWeightRegular FontWeight = "regular"
	FontWeightBold    FontWeight = "bold"
)

type Font struct {
	Name   string
	Size   float64
	Color  string
	Weight FontWeight
}

// Get font weight of canvas type
func (f *Font) GetFontStyle() canvas.FontStyle {
	switch f.Weight {
	case FontWeightRegular:
		return canvas.FontRegular
	case FontWeightBold:
		return canvas.FontBold
	default:
		return canvas.FontRegular
	}
}

type FontMetadata struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func getFontMetadataByPath(fontPath string) (*FontMetadata, error) {
	fontBytes, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	font, err := sfnt.Parse(fontBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing font: %w", err)
	}

	name, err := font.Name(nil, sfnt.NameIDFamily)
	if err != nil {
		return nil, fmt.Errorf("retrieving font name: %w", err)
	}

	return &FontMetadata{
		Name: name,
		Path: fontPath,
	}, nil
}

// Scan through the directory to process .ttf and .otf files.
func ScanFontDir(dir string) ([]FontMetadata, error) {
	var fonts []FontMetadata

	err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(info.Name()))
		if ext != ".ttf" && ext != ".otf" {
			return nil
		}

		meta, err := getFontMetadataByPath(path)
		if err != nil {
			log.Printf("Skipping %q: %v", path, err)
			return nil
		}

		fonts = append(fonts, *meta)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return fonts, nil
}

// List the available font family and its path
func GetAvailableFonts(path string) ([]*FontMetadata, error) {
	var fonts []*FontMetadata

	if path == "" {
		path = "font_metadata.json"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fonts, fmt.Errorf("Error reading font_metadata.json: %v", err)
	}

	if err := json.Unmarshal(data, &fonts); err != nil {
		return fonts, fmt.Errorf("Error unmarshalling JSON: %v", err)
	}

	return fonts, nil
}

type FontLoader struct {
	Cfg            *Config
	AvailableFonts []*FontMetadata
}

func NewFontLoader(cfg *Config) *FontLoader {
	// Load the font metadata from the JSON file
	fonts, err := GetAvailableFonts(cfg.FontMetadataPath)
	if err != nil {
		log.Fatalf("failed to load font metadata: %v", err)
	}

	return &FontLoader{
		Cfg:            cfg,
		AvailableFonts: fonts,
	}
}

func (fl *FontLoader) GetAvailableFontMetadataByName(fontName string) (*FontMetadata, error) {
	for _, font := range fl.AvailableFonts {
		if font.Name == fontName {
			return font, nil
		}
	}
	return nil, fmt.Errorf("font %s not found", fontName)
}

func (fl *FontLoader) LoadFont(fontName string, fontStyle canvas.FontStyle) (*canvas.FontFamily, error) {
	fontMetadata, err := fl.GetAvailableFontMetadataByName(fontName)
	if err != nil {
		return nil, fmt.Errorf("failed to get font metadata: %v", err)
	}

	if fontMetadata == nil {
		return nil, fmt.Errorf("font metadata is nil")
	}

	fontFamily := canvas.NewFontFamily(fontMetadata.Name)
	err = fontFamily.LoadFontFile(fontMetadata.Path, fontStyle)
	if err != nil {
		return nil, fmt.Errorf("failed to load font file: %v", err)
	}

	// TODO: load fallback font
	// err := fontFamily.LoadFontFile(FallBackFont, canvas.FontRegular)
	// if err != nil {
	// 	log.Fatalf("failed to load fallback font: %v", err)
	// }

	return fontFamily, nil
}
