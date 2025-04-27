package autocert

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type Settings struct {
	RemoveLineBreaksBool bool
	EmbedQRCode          bool
}

func NewDefaultSettings() *Settings {
	return &Settings{
		RemoveLineBreaksBool: true,
		EmbedQRCode:          true,
	}
}

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

// Returns the output directory path for this generator.
func (cg *CertificateGenerator) GetOutputDir() string {
	outputDir := filepath.Join(cg.Cfg.OutputDir, cg.ID)
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			panic(fmt.Sprintf("failed to create output directory: %s", err))
		}
	}
	return outputDir
}

// Returns the temporary directory path for this generator.
func (cg *CertificateGenerator) GetTmpDir() string {
	tmpDir := filepath.Join(cg.Cfg.TmpDir, cg.ID)
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			panic(fmt.Sprintf("failed to create tmp directory: %s", err))
		}
	}
	return tmpDir
}

// Apply all signature annotations to the input PDF file.
func (cg *CertificateGenerator) applySignatures(inputFile string) (string, error) {
	currentFile := inputFile

	for page, sigAnnots := range cg.Annotations.PageSignatureAnnotations {
		for _, annot := range sigAnnots {
			tmpOut, err := os.CreateTemp(cg.GetTmpDir(), "autocert_*.pdf")
			if err != nil {
				return "", err
			}

			selectedPages := []string{fmt.Sprintf("%d", page)}
			signatureFile := annot.SignatureFilePath

			// Skip if the signature file does not exist
			if _, err := os.Stat(signatureFile); os.IsNotExist(err) {
				log.Printf("Signature file %s does not exist, skipping annotation %s\n", signatureFile, annot.ID)
				continue
			}

			signatureFile, err = cg.normalizeSignatureFormat(signatureFile, annot)
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

func (cg *CertificateGenerator) normalizeSignatureFormat(signatureFile string, annot SignatureAnnotate) (string, error) {
	switch filepath.Ext(signatureFile) {
	case ".png", ".jpg", ".jpeg":
		// Resize the image following the annotation size
		tmpImg, err := os.CreateTemp(cg.GetTmpDir(), "autocert_img_*.png")
		if err != nil {
			return "", fmt.Errorf("failed to create temporary image file: %w", err)
		}

		if err := ResizeImage(signatureFile, tmpImg.Name(), annot.Width, annot.Height); err != nil {
			return "", fmt.Errorf("failed to resize image for annotation %s: %w", annot.ID, err)
		}

		return tmpImg.Name(), nil
	case ".svg":
		tmpSvg, err := os.CreateTemp(cg.GetTmpDir(), "autocert_svg_sig_*.pdf")
		if err != nil {
			return "", fmt.Errorf("failed to create temporary SVG file: %w", err)
		}

		if err := SvgToPdf(signatureFile, tmpSvg.Name(), annot.Width, annot.Height); err != nil {
			return "", fmt.Errorf("failed to convert SVG to PDF for annotation %s: %w", annot.ID, err)
		}

		return tmpSvg.Name(), nil
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
	tmpOut, err := os.CreateTemp(dir, "autocert_*.pdf")
	if err != nil {
		return "", err
	}

	// Create temporary text file for rendering the text
	txtFile, err := os.CreateTemp(dir, "autocert_svg_text_*.pdf")
	if err != nil {
		return "", err
	}

	textRenderer, err := NewTextRenderer(cg.Cfg, *annot.Rect(), *annot.Font(), cg.Settings)
	if err != nil {
		return "", err
	}

	if err := textRenderer.RenderSvgTextAsPdf(annot.Value, annot.TextAlign, txtFile.Name()); err != nil {
		return "", err
	}

	if err := ApplyWatermarkToPdf(currentFile, tmpOut.Name(), selectedPages, txtFile.Name(), annot.X, annot.Y); err != nil {
		return "", err
	}

	return tmpOut.Name(), nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return nil
}

// job represents a single certificate generation job.
type Job struct {
	// CSV row index
	index int
	// CSV data for this row.
	data map[string]string
	// Temporary directory for this worker
	tmpDir string
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

// Generate certificate based on the provided template and CSV data and annotations type.
// Returns a list of generated certificate file paths or each row in the CSV.
// If no CSV is provided, it will generate a single certificate and apply the annotations.
// The output file names will be based on the provided pattern. Eg. "certificate_%d.pdf"
func (cg *CertificateGenerator) Generate(outputFilePattern string) ([]string, error) {
	defer os.RemoveAll(cg.GetTmpDir())

	baseFile, err := cg.applySignatures(cg.TemplatePath)
	if err != nil {
		return nil, err
	}

	// Handle case with no CSV file, just generate a single certificate
	if cg.CSVPath == "" {
		outputFile := filepath.Join(cg.GetOutputDir(), fmt.Sprintf(outputFilePattern, 1))
		if err := os.Rename(baseFile, outputFile); err != nil {
			return nil, err
		}
		return []string{outputFile}, nil
	}

	return cg.generateFromCSV(baseFile, outputFilePattern)
}

// Handles the parallel generation of certificates from CSV data.
func (cg *CertificateGenerator) generateFromCSV(baseFile, outputFilePattern string) ([]string, error) {
	dataMaps, err := cg.readCSVData()
	if err != nil {
		return nil, err
	}

	maxWorkers := determineWorkerCount(len(dataMaps))

	// Create channels for job distribution and result collection
	jobs := make(chan Job, len(dataMaps))
	results := make(chan Result, len(dataMaps))

	var wg sync.WaitGroup
	for range maxWorkers {
		wg.Add(1)
		go cg.certificateWorker(jobs, results, baseFile, &wg)
	}

	for i, row := range dataMaps {
		// Create temp worker-specific directory for each job
		workerID := fmt.Sprintf("worker-%d", i)
		workerTmpDir := filepath.Join(cg.GetTmpDir(), workerID)
		if err := os.MkdirAll(workerTmpDir, 0755); err != nil {
			results <- Result{index: i, outputFile: "", err: fmt.Errorf("failed to create worker tmp dir: %w", err)}
			continue
		}

		jobs <- Job{index: i, data: row, tmpDir: workerTmpDir}
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	return cg.collectResults(results, len(dataMaps))
}

func (cg *CertificateGenerator) readCSVData() ([]map[string]string, error) {
	records, err := ReadCSVFromFile(cg.CSVPath)
	if err != nil {
		return nil, err
	}

	dataMaps, err := ParseCSVToMap(records)
	if err != nil {
		return nil, err
	}

	return dataMaps, nil
}

// It defaults to the value of runtime.NumCPU (core count)
// Note: Can change to more than core count if needed
// max worker should be at least 1 and should not exceed job count
func determineWorkerCount(jobCount int) int {
	maxWorkers := min(max(runtime.GOMAXPROCS(0), 1), jobCount)

	fmt.Printf("Using %d workers for processing\n", maxWorkers)

	return maxWorkers
}

// Process certificate generation jobs.
func (cg *CertificateGenerator) certificateWorker(jobs <-chan Job, results chan<- Result, baseFile string, wg *sync.WaitGroup) {
	defer wg.Done()

	for j := range jobs {
		outputFile, err := cg.processJob(j, baseFile)
		results <- Result{j.index, outputFile, err}

		// Clean up worker directory after processing
		os.RemoveAll(j.tmpDir)
	}
}

// Handle the processing of a single certificate job.
func (cg *CertificateGenerator) processJob(j Job, baseFile string) (string, error) {
	// Copy the base file for this worker
	workerBaseFile := filepath.Join(j.tmpDir, "base.pdf")
	if err := copyFile(baseFile, workerBaseFile); err != nil {
		return "", err
	}

	currentFile := workerBaseFile

	for page, colAnnots := range cg.Annotations.PageColumnAnnotations {
		for _, annot := range colAnnots {
			// Substitute the value from CSV data
			modifiedAnnot := annot
			modifiedAnnot.Value = j.data[annot.Value]

			var err error
			currentFile, err = cg.applyTextAnnotation(currentFile, page, modifiedAnnot, j.tmpDir)
			if err != nil {
				return "", fmt.Errorf("failed to apply text annotation on page %d for row %d: %w", page, j.index, err)
			}
		}
	}

	// After applying all annotations, we can finalize the PDF and embed the QR code if enabled
	if cg.Settings.EmbedQRCode {
		tmpQrCodeFile, err := os.CreateTemp(j.tmpDir, "autocert_qr_*.pdf")
		if err != nil {
			return "", err
		}
		defer os.Remove(tmpQrCodeFile.Name())

		// TODO: put actual qr code link
		GenerateQRCodeAsPdf(fmt.Sprintf("www.youtube.com?workerIndex=%d", j.index+1), tmpQrCodeFile.Name(), 50)
		err = EmbedQRCodeToPdf(currentFile, currentFile, tmpQrCodeFile.Name(), []string{})
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
func (cg *CertificateGenerator) collectResults(results <-chan Result, totalCount int) ([]string, error) {
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
	for i := range totalCount {
		if file, ok := resultMap[i]; ok {
			generatedFiles = append(generatedFiles, file)
		} else {
			return nil, fmt.Errorf("missing result for row %d", i)
		}
	}

	return generatedFiles, nil
}
