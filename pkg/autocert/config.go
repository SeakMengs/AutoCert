package autocert

import (
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	// A path to json where it store font name and path to the font file
	FontMetadataPath string
	// Directory where the output files are stored after processing
	OutputDir string
	// Directory where the temporary files are stored during processing, the file will be deleted after processing
	TmpDir string
	// Directory where the state files are stored, which will be used to store the state of the project
	StateDir string
	// FallbackFont is the font that will be used if the requested font is not available in the FontDir
	FallbackFont string
}

func NewDefaultConfig() Config {
	cfg := Config{
		FontMetadataPath: "font_metadata.json",
		OutputDir:        "autocert_tmp/output",
		TmpDir:           "autocert_tmp/tmp",
		StateDir:         "autocert_tmp/state",
		FallbackFont:     "Arial",
	}

	// Create the directories if they do not exist
	// 0755 mean owner can read, write and execute
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
	}
	if err := os.MkdirAll(cfg.TmpDir, 0755); err != nil {
		fmt.Printf("Error creating tmp directory: %v\n", err)
	}
	if err := os.MkdirAll(cfg.StateDir, 0755); err != nil {
		fmt.Printf("Error creating state directory: %v\n", err)
	}

	return cfg
}

/*
 * StatePath returns the json path to the state file for a given project ID.
 * The state file is used to store the state of the project, such that it can continue
 * from where it left off in case of a crash or restart.
 */
func (c *Config) StatePath(projectID string) string {
	return filepath.Join(c.StateDir, projectID+".json")
}
