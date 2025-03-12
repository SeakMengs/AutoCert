package autocert

import (
	"fmt"

	"github.com/h2non/bimg"
)

// Require libvips to be installed on the system.
func ResizeImage(in, out string, width, height int) error {
	buffer, err := bimg.Read(in)
	if err != nil {
		return fmt.Errorf("failed to read image: %v", err)
	}

	options := bimg.Options{
		Width:        width,
		Height:       height,
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

	err = bimg.Write(out, newImage)
	if err != nil {
		return fmt.Errorf("failed to write image: %v", err)
	}

	return nil
}
