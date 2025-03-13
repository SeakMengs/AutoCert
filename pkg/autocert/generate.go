package autocert

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

// Settings holds options for rendering text.
type Settings struct {
	TextFitRectBox       bool
	RemoveLineBreaksBool bool
	EmbedQRCode          bool
}

// CertificateGenerator holds the necessary state for certificate generation.
type CertificateGenerator struct {
	// ID is a unique identifier which will be used to create folder and store the generated files.
	ID           string
	TemplatePath string
	CSVPath      string
	Cfg          Config
	Annotations  PageAnnotations
	Settings     Settings
}

func NewCertificateGenerator(id, templatePath, csvPath string, cfg Config, annotations PageAnnotations, settings Settings) *CertificateGenerator {
	return &CertificateGenerator{
		ID:           id,
		TemplatePath: templatePath,
		CSVPath:      csvPath,
		Cfg:          cfg,
		Annotations:  annotations,
		Settings:     settings,
	}
}

// GetOutputDir returns the output directory path for this generator.
func (cg *CertificateGenerator) GetOutputDir() string {
	outputDir := filepath.Join(cg.Cfg.OutputDir, cg.ID)
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			panic(fmt.Sprintf("failed to create output directory: %s", err))
		}
	}
	return outputDir
}

// GetTmpDir returns the temporary directory path for this generator.
func (cg *CertificateGenerator) GetTmpDir() string {
	tmpDir := filepath.Join(cg.Cfg.TmpDir, cg.ID)
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			panic(fmt.Sprintf("failed to create tmp directory: %s", err))
		}
	}
	return tmpDir
}

// GetStateDir returns the state directory path for this generator.
func (cg *CertificateGenerator) GetStateDir() string {
	stateDir := filepath.Join(cg.Cfg.StateDir, cg.ID)
	if _, err := os.Stat(stateDir); os.IsNotExist(err) {
		if err := os.MkdirAll(stateDir, 0755); err != nil {
			panic(fmt.Sprintf("failed to create state directory: %s", err))
		}
	}
	return stateDir
}

// Apply all signature annotations to the input PDF file.
func (cg *CertificateGenerator) applySignatures(inputFile string) (string, error) {
	currentFile := inputFile

	for page, sigAnnots := range cg.Annotations.PageSignatureAnnotations {
		for _, annot := range sigAnnots {
			tmpOut, err := os.CreateTemp(cg.GetTmpDir(), "autocert-*.pdf")
			if err != nil {
				return "", fmt.Errorf("failed to create temporary file to apply signatures: %w", err)
			}

			selectedPages := []string{fmt.Sprintf("%d", page)}
			var signatureFile = annot.SignatureFilePath

			// Skip if the signature file does not exist
			if _, err := os.Stat(signatureFile); os.IsNotExist(err) {
				continue
			}

			signatureFile, err = cg.prepareSignatureFile(signatureFile, annot)
			if err != nil {
				return "", err
			}

			if err := ApplyWatermarkToPdf(currentFile, tmpOut.Name(), selectedPages, signatureFile, annot.X, annot.Y); err != nil {
				return "", fmt.Errorf("failed to apply signature watermark for annotation %s: %w", annot.ID, err)
			}

			currentFile = tmpOut.Name()
		}
	}

	return currentFile, nil
}

// prepareSignatureFile prepares the signature file based on its type.
func (cg *CertificateGenerator) prepareSignatureFile(signatureFile string, annot SignatureAnnotate) (string, error) {
	switch filepath.Ext(signatureFile) {
	case ".png", ".jpg", ".jpeg":
		// Resize the image following the annotation size
		tmpImg, err := os.CreateTemp(cg.GetTmpDir(), "autocert_img-*.png")
		if err != nil {
			return "", fmt.Errorf("failed to create temporary image file: %w", err)
		}

		if err := ResizeImage(signatureFile, tmpImg.Name(), annot.Width, annot.Height); err != nil {
			return "", fmt.Errorf("failed to resize image for annotation %s: %w", annot.ID, err)
		}

		return tmpImg.Name(), nil

	case ".svg":
		// TODO: Handle SVG signature files convert to pdf
		return signatureFile, nil

	case ".pdf":
		return signatureFile, nil

	default:
		return "", fmt.Errorf("unsupported signature file type: %s", filepath.Ext(signatureFile))
	}
}

// Apply all text annotations to the input PDF file.
func (cg *CertificateGenerator) applyTextAnnotation(currentFile string, page uint, annot ColumnAnnotate, tmpDir string) (string, error) {
	selectedPages := []string{fmt.Sprintf("%d", page)}

	// Create temporary files in the provided directory (tpmDir is worker pool dir)
	dir := tmpDir
	if dir == "" {
		dir = cg.GetTmpDir()
	}

	// Create temporary output file
	tmpOut, err := os.CreateTemp(dir, "autocert-*.pdf")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}

	// Create temporary text file for rendering the text
	txtFile, err := os.CreateTemp(dir, "autocert_svg_text-*.pdf")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary text file: %w", err)
	}

	rect := Rect{
		Width:  annot.Width,
		Height: annot.Height,
	}

	font := Font{
		Name:   annot.FontName,
		Size:   annot.FontSize,
		Color:  annot.FontColor,
		Weight: annot.FontWeight,
	}

	// Render text and apply as watermark
	textRenderer := NewTextRenderer(cg.Cfg, rect, font, cg.Settings)
	if err := textRenderer.RenderSvgTextAsPdf(annot.Value, TextAlignCenter, txtFile.Name()); err != nil {
		return "", fmt.Errorf("failed to render text for annotation %s: %w", annot.ID, err)
	}

	if err := ApplyWatermarkToPdf(currentFile, tmpOut.Name(), selectedPages, txtFile.Name(), annot.X, annot.Y); err != nil {
		return "", fmt.Errorf("failed to apply text watermark for annotation %s: %w", annot.ID, err)
	}

	return tmpOut.Name(), nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	return nil
}

// job represents a single certificate generation job.
type Job struct {
	// CSV row index
	index int
	// CSV data for this row.
	data map[string]string
}

// result represents the result of a certificate generation job.
type Result struct {
	// CSV row index
	index int
	// Output file path after worker processing
	outputFile string
	// Error if any occurred during processing
	// (nil if processing was successful)
	err error
}

func (cg *CertificateGenerator) Generate(outputFilePattern string) ([]string, error) {
	// Clean up temporary files when done
	defer os.RemoveAll(cg.GetTmpDir())

	// Apply signatures to the base template
	baseFile, err := cg.applySignatures(cg.TemplatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to apply signatures: %w", err)
	}

	// Handle case with no CSV file
	if cg.CSVPath == "" {
		outputFile := filepath.Join(cg.GetOutputDir(), fmt.Sprintf(outputFilePattern, 1))
		if err := os.Rename(baseFile, outputFile); err != nil {
			return nil, fmt.Errorf("failed to finalize certificate: %w", err)
		}
		return []string{outputFile}, nil
	}

	// Process CSV data in parallel
	return cg.generateFromCSV(baseFile, outputFilePattern)
}

// Handles the parallel generation of certificates from CSV data.
func (cg *CertificateGenerator) generateFromCSV(baseFile, outputFilePattern string) ([]string, error) {
	// Read and parse CSV data
	dataMaps, err := cg.readCSVData()
	if err != nil {
		return nil, err
	}

	maxWorkers := determineWorkerCount(len(dataMaps))

	// Create channels for job distribution and result collection
	jobs := make(chan Job, len(dataMaps))
	results := make(chan Result, len(dataMaps))

	// Start worker pool
	var wg sync.WaitGroup
	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go cg.certificateWorker(jobs, results, baseFile, &wg)
	}

	// Send jobs to workers
	for i, row := range dataMaps {
		jobs <- Job{index: i, data: row}
	}
	close(jobs)

	// Wait for all workers to finish and close the results channel
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and organize results
	return cg.collectResults(results, len(dataMaps), outputFilePattern)
}

// readCSVData reads and parses the CSV file.
func (cg *CertificateGenerator) readCSVData() ([]map[string]string, error) {
	records, err := ReadCSV(cg.CSVPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	dataMaps, err := ParseCSVToMap(records)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	return dataMaps, nil
}

// Find optimal number of worker goroutines.
func determineWorkerCount(jobCount int) int {
	// Get the number of available CPU cores
	maxWorkers := runtime.GOMAXPROCS(0)

	if maxWorkers > 8 {
		maxWorkers = 8
	}

	if maxWorkers > jobCount {
		maxWorkers = jobCount
	}

	return maxWorkers
}

// Process certificate generation jobs.
func (cg *CertificateGenerator) certificateWorker(jobs <-chan Job, results chan<- Result, baseFile string, wg *sync.WaitGroup) {
	defer wg.Done()

	for j := range jobs {
		// Create worker-specific directory
		workerID := fmt.Sprintf("worker-%d", j.index)
		workerTmpDir := filepath.Join(cg.GetTmpDir(), workerID)

		if err := os.MkdirAll(workerTmpDir, 0755); err != nil {
			results <- Result{j.index, "", fmt.Errorf("failed to create worker tmp dir: %w", err)}
			continue
		}

		// Process the job and send result
		outputFile, err := cg.processJob(j, workerTmpDir, baseFile)
		results <- Result{j.index, outputFile, err}

		// Clean up worker directory after processing
		os.RemoveAll(workerTmpDir)
	}
}

// Handle the processing of a single certificate job.
func (cg *CertificateGenerator) processJob(j Job, workerTmpDir, baseFile string) (string, error) {
	// Copy the base file for this worker
	workerBaseFile := filepath.Join(workerTmpDir, "base.pdf")
	if err := copyFile(baseFile, workerBaseFile); err != nil {
		return "", fmt.Errorf("failed to copy base file: %w", err)
	}

	// Apply text annotations
	currentFile := workerBaseFile

	for page, colAnnots := range cg.Annotations.PageColumnAnnotations {
		for _, annot := range colAnnots {
			// Substitute the value from CSV data
			modifiedAnnot := annot
			modifiedAnnot.Value = j.data[annot.Value]

			var err error
			currentFile, err = cg.applyTextAnnotation(currentFile, page, modifiedAnnot, workerTmpDir)
			if err != nil {
				return "", fmt.Errorf("failed to apply text annotation on page %d for row %d: %w", page, j.index, err)
			}
		}
	}

	// After applying all annotations, we can finalize the PDF and embed the QR code if enabled
	if cg.Settings.EmbedQRCode {
		tmpQrCodeFile := filepath.Join(workerTmpDir, fmt.Sprintf("qr_code_%d.png", j.index))

		// TODO: put actual qr code link
		GenerateQRCode(fmt.Sprintf("www.youtube.com?workerIndex=%d", j.index+1), tmpQrCodeFile, 50)
		err := EmbedQRCodeToPdf(currentFile, currentFile, tmpQrCodeFile, []string{})
		if err != nil {
			return "", fmt.Errorf("failed to embed QR code for row %d: %w", j.index, err)
		}
	}

	// Move the final output file to the output directory
	outputFile := filepath.Join(cg.GetOutputDir(), fmt.Sprintf("certificate_%d.pdf", j.index+1))
	if err := os.Rename(currentFile, outputFile); err != nil {
		return "", fmt.Errorf("failed to finalize certificate for row %d: %w", j.index, err)
	}

	return outputFile, nil
}

// Collect and organize the results from worker goroutines.
func (cg *CertificateGenerator) collectResults(results <-chan Result, totalCount int, outputFilePattern string) ([]string, error) {
	// A map of generated files indexed by their original CSV row index
	// This is used to ensure the output files are in the correct order
	resultMap := make(map[int]string)
	var firstErr error

	for r := range results {
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
		} else {
			resultMap[r.index] = r.outputFile
		}
	}

	if firstErr != nil {
		return nil, firstErr
	}

	// Build result list in the correct order
	generatedFiles := make([]string, 0, totalCount)
	for i := 0; i < totalCount; i++ {
		if file, ok := resultMap[i]; ok {
			generatedFiles = append(generatedFiles, file)
		} else {
			return nil, fmt.Errorf("missing result for row %d", i)
		}
	}

	return generatedFiles, nil
}
