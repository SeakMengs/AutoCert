package autocert

import (
	"fmt"
	"testing"

	"github.com/tdewolff/canvas"
)

func TestFontLoader(t *testing.T) {
	testPath := "../../"
	fontMetaPath := fmt.Sprintf("%s%s", testPath, "font_metadata.json")
	fontMetadata, err := GetAvailableFonts(fontMetaPath)
	if err != nil {
		t.Fatalf("Failed to get available fonts: %v", err)
	}
	if len(fontMetadata) == 0 {
		t.Skip("No fonts available in font_metadata.json to test LoadFont")
	}

	cfg := Config{FontMetadataPath: fontMetaPath}
	fontLoader, err := NewFontLoader(cfg)
	if err != nil {
		t.Fatalf("Failed to create FontLoader: %v", err)
	}

	for _, meta := range fontLoader.AvailableFonts {
		t.Run(meta.Name, func(t *testing.T) {
			fontFamily, err := fontLoader.LoadFont(meta.Name, canvas.FontRegular)
			if err != nil {
				t.Errorf("LoadFont failed for %s: %v", meta.Name, err)
			}
			if fontFamily == nil {
				t.Errorf("LoadFont returned nil fontFamily for %s", meta.Name)
			}
		})
	}
}
