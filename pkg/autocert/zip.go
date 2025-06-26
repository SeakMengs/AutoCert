package autocert

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

func addFileToZip(archive *zip.Writer, filePath, archivePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return nil // Skip directories
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = archivePath
	header.Method = zip.Deflate

	writer, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}

	fileReader, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer fileReader.Close()

	_, err = io.Copy(writer, fileReader)
	return err
}

func createZipArchive(zipFile string) (*os.File, *zip.Writer, error) {
	zipWriter, err := os.Create(zipFile)
	if err != nil {
		return nil, nil, err
	}

	archive := zip.NewWriter(zipWriter)
	return zipWriter, archive, nil
}

func ZipDir(dir string, zipFile string) error {
	zipWriter, archive, err := createZipArchive(zipFile)
	if err != nil {
		return err
	}
	defer zipWriter.Close()
	defer archive.Close()

	return filepath.Walk(dir, func(filePath string, info os.FileInfo, err error) error {
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

		return addFileToZip(archive, filePath, relPath)
	})
}

func ZipFiles(inFiles []string, zipFile string) error {
	zipWriter, archive, err := createZipArchive(zipFile)
	if err != nil {
		return err
	}
	defer zipWriter.Close()
	defer archive.Close()

	for _, filePath := range inFiles {
		if err := addFileToZip(archive, filePath, filepath.Base(filePath)); err != nil {
			return err
		}
	}

	return nil
}
