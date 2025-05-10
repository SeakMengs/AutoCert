package util

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

func ZipDir(dir string, zipFile string) error {
	zipWriter, err := os.Create(zipFile)
	if err != nil {
		return err
	}
	defer zipWriter.Close()

	archive := zip.NewWriter(zipWriter)
	defer archive.Close()

	err = filepath.Walk(dir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the zip file itself to avoid recursion
		if filePath == zipFile {
			return nil
		}

		relPath, err := filepath.Rel(dir, filePath)
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip directories
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		fileReader, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer fileReader.Close()

		if _, err := io.Copy(writer, fileReader); err != nil {
			return err
		}

		return nil
	})

	return err
}
