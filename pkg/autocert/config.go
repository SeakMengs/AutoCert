package autocert

import (
	"fmt"
	"os"
)

type Config struct {
	// A path to json where it store font name and path to the font file
	FontMetadataPath string
	// Directory where the output files are stored after processing
	OutputDir string
	// Directory where the temporary files are stored during processing, the file will be deleted after processing
	TmpDir string
}

func NewDefaultConfig() *Config {
	cfg := Config{
		FontMetadataPath: "font_metadata.json",
		OutputDir:        fmt.Sprintf("%s/autocert/generate/output", os.TempDir()),
		TmpDir:           fmt.Sprintf("%s/autocert/generate/tmp", os.TempDir()),
	}

	// Create the directories if they do not exist
	// 0755 mean owner can read, write and execute
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
	}
	if err := os.MkdirAll(cfg.TmpDir, 0755); err != nil {
		fmt.Printf("Error creating tmp directory: %v\n", err)
	}

	return &cfg
}
