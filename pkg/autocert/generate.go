package autocert

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type CertificateType int

const (
	CertificateTypeNormal CertificateType = iota
	CertificateTypeMerged
	CertificateTypeZip
)

type GeneratedResult struct {
	Number   int
	FilePath string
	FileName string
	ID       string
	Type     CertificateType
}

type ProgressInfo struct {
	Generated    int           `json:"generated"`
	Total        int           `json:"total"`
	Percentage   float64       `json:"percentage"`
	TimeElapsed  time.Duration `json:"time_elapsed"`
	TimeLeft     time.Duration `json:"time_left"`
	CurrentPhase string        `json:"current_phase"`
	EstimatedETA time.Time     `json:"estimated_eta"`
}

type ProgressCallback func(progress ProgressInfo)

type Settings struct {
	RemoveLineBreaksBool bool
	EmbedQRCode          bool
	QrURLPattern         string
	MergeAfterGenerate   bool
	ZipAfterGenerate     bool
	ProgressCallback     ProgressCallback
}

func NewDefaultSettings(qrUrlPattern string) *Settings {
	return &Settings{
		RemoveLineBreaksBool: true,
		EmbedQRCode:          true,
		QrURLPattern:         qrUrlPattern,
		MergeAfterGenerate:   true,
		ZipAfterGenerate:     true,
		// Default to no callback
		ProgressCallback: nil,
	}
}

type CertificateGenerator struct {
	ID           string
	TemplatePath string
	CSVPath      string
	Cfg          Config
	Annotations  PageAnnotations
	Settings     Settings
	// Eg: "certificate_%s"
	OutFilePattern string
	csvData        []map[string]string
	textRenderers  map[string]*TextRenderer

	// Progress tracking fields
	startTime      time.Time
	completedCount int64
	totalCount     int
	progressMutex  sync.RWMutex
}

func NewCertificateGenerator(id, templatePath, csvPath string, cfg Config, annotations PageAnnotations, settings Settings, outFilePattern string) *CertificateGenerator {
	return &CertificateGenerator{
		ID:             id,
		TemplatePath:   templatePath,
		CSVPath:        csvPath,
		Cfg:            cfg,
		Annotations:    annotations,
		Settings:       settings,
		OutFilePattern: outFilePattern,
		textRenderers:  make(map[string]*TextRenderer),
	}
}

func (cg *CertificateGenerator) initializeProgress() {
	cg.startTime = time.Now()
	atomic.StoreInt64(&cg.completedCount, 0)
	cg.totalCount = 0
	cg.updateProgress("Initialization")
}

func (cg *CertificateGenerator) updateProgress(phase string) {
	if cg.Settings.ProgressCallback == nil {
		return
	}

	cg.progressMutex.RLock()
	completed := atomic.LoadInt64(&cg.completedCount)
	total := int64(cg.totalCount)
	elapsed := time.Since(cg.startTime)
	cg.progressMutex.RUnlock()

	var percentage float64
	var timeLeft time.Duration
	var eta time.Time

	if total > 0 {
		percentage = float64(completed) / float64(total) * 100

		if completed > 0 {
			avgTimePerCert := elapsed / time.Duration(completed)
			remaining := total - completed
			timeLeft = avgTimePerCert * time.Duration(remaining)
			eta = time.Now().Add(timeLeft)
		}
	}

	progress := ProgressInfo{
		Generated:    int(completed),
		Total:        int(total),
		Percentage:   percentage,
		TimeElapsed:  elapsed,
		TimeLeft:     timeLeft,
		CurrentPhase: phase,
		EstimatedETA: eta,
	}

	// Call the callback in a separate goroutine to avoid blocking
	go cg.Settings.ProgressCallback(progress)
}

func (cg *CertificateGenerator) incrementProgress() {
	atomic.AddInt64(&cg.completedCount, 1)
	cg.updateProgress("Generating certificates")
}

func (cg *CertificateGenerator) OutputDir() string {
	outputDir := filepath.Join(cg.Cfg.OutputDir, cg.ID)
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			panic(fmt.Sprintf("failed to create output directory: %s", err))
		}
	}
	return outputDir
}

func (cg *CertificateGenerator) TempDir() string {
	tmpDir := filepath.Join(cg.Cfg.TmpDir, cg.ID)
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			panic(fmt.Sprintf("failed to create tmp directory: %s", err))
		}
	}
	return tmpDir
}

func (cg *CertificateGenerator) embedSignatures(inputFile string) (string, error) {
	currentFile := inputFile

	for page, sigAnnots := range cg.Annotations.PageSignatureAnnotations {
		for _, annot := range sigAnnots {
			tmpOut, err := os.CreateTemp(cg.TempDir(), "autocert_*.pdf")
			if err != nil {
				return "", err
			}

			selectedPages := []string{fmt.Sprintf("%d", page)}
			signatureFile := annot.SignatureFilePath

			if _, err := os.Stat(signatureFile); os.IsNotExist(err) {
				log.Printf("Signature file %s does not exist, skipping annotation %s\n", signatureFile, annot.ID)
				continue
			}

			signatureFile, err = cg.convertSignatureFormat(signatureFile, annot)
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

func (cg *CertificateGenerator) convertSignatureFormat(signatureFile string, annot SignatureAnnotate) (string, error) {
	switch filepath.Ext(signatureFile) {
	case ".png", ".jpg", ".jpeg":
		tmpImg, err := os.CreateTemp(cg.TempDir(), "autocert_img_*.png")
		if err != nil {
			return "", fmt.Errorf("failed to create temporary image file: %w", err)
		}

		if err := ResizeImage(signatureFile, tmpImg.Name(), annot.Width, annot.Height, true); err != nil {
			return "", fmt.Errorf("failed to resize image for annotation %s: %w", annot.ID, err)
		}

		return tmpImg.Name(), nil
	case ".svg":
		tmpSvg, err := os.CreateTemp(cg.TempDir(), "autocert_svg_sig_*.pdf")
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

func (cg *CertificateGenerator) initializeTextRenderers() error {
	for _, colAnnots := range cg.Annotations.PageColumnAnnotations {
		for _, annot := range colAnnots {
			if _, exists := cg.textRenderers[annot.ID]; exists {
				continue
			}

			font := annot.Font()
			if annot.TextFitRectBox {
				font.Size = 0
			}

			textRenderer, err := NewTextRenderer(cg.Cfg, *annot.Rect(), *font, cg.Settings)
			if err != nil {
				return fmt.Errorf("failed to create text renderer for annotation %s: %w", annot.ID, err)
			}

			cg.textRenderers[annot.ID] = textRenderer
		}
	}
	return nil
}

func (cg *CertificateGenerator) embedTextAnnotation(currentFile string, page uint, annot ColumnAnnotate, tmpDir string) (string, error) {
	selectedPages := []string{fmt.Sprintf("%d", page)}

	dir := tmpDir
	if dir == "" {
		dir = cg.TempDir()
	}

	tmpOut, err := os.CreateTemp(dir, "autocert_temp_template_pdf_*.pdf")
	if err != nil {
		return "", err
	}

	txtFile, err := os.CreateTemp(dir, "autocert_svg_text_*.pdf")
	if err != nil {
		return "", err
	}

	textRenderer := cg.textRenderers[annot.ID]

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
	return err
}

type generationJob struct {
	index  int
	data   map[string]string
	tmpDir string
}

type generationResult struct {
	id         string
	index      int
	outputFile string
	err        error
}

func (cg *CertificateGenerator) Generate() ([]GeneratedResult, error) {
	cg.initializeProgress()

	defer os.RemoveAll(cg.TempDir())

	cg.updateProgress("Preparing template")

	baseFile, err := cg.embedSignatures(cg.TemplatePath)
	if err != nil {
		return nil, err
	}

	cg.updateProgress("Loading CSV data")

	csvDataMaps, err := cg.loadCSVData()
	if err != nil {
		return nil, err
	}
	cg.csvData = csvDataMaps

	cg.totalCount = len(cg.csvData)
	if cg.totalCount == 0 {
		// For single certificate generation
		cg.totalCount = 1
	}

	if len(cg.csvData) == 0 || len(cg.Annotations.PageColumnAnnotations) == 0 {
		return cg.generateSingleCertificate(baseFile)
	}

	cg.updateProgress("Initializing text renderers")

	if err := cg.initializeTextRenderers(); err != nil {
		return nil, err
	}

	return cg.generateBatchCertificates(baseFile)
}

func (cg *CertificateGenerator) generateSingleCertificate(baseFile string) ([]GeneratedResult, error) {
	cg.updateProgress("Generating single certificate")

	outputFile := filepath.Join(cg.OutputDir(), fmt.Sprintf(cg.OutFilePattern, "1")+".pdf")

	// Use copy instead of os.Rename to avoid invalid cross-device link
	if err := copyFile(baseFile, outputFile); err != nil {
		return nil, err
	}
	os.Remove(baseFile)

	cg.incrementProgress()

	results := make(chan generationResult, 1)
	results <- generationResult{
		id:         uuid.NewString(),
		index:      0,
		outputFile: outputFile,
		err:        nil,
	}
	close(results)

	return cg.aggregateResults(results, 1)
}

func (cg *CertificateGenerator) generateBatchCertificates(baseFile string) ([]GeneratedResult, error) {
	maxWorkers := DeterminWorkers(len(cg.csvData))
	fmt.Printf("Using %d workers for generating certificate for project id: %s\n", maxWorkers, cg.ID)

	cg.updateProgress("Starting batch generation")

	jobs := make(chan generationJob, len(cg.csvData))
	results := make(chan generationResult, len(cg.csvData))

	var wg sync.WaitGroup
	for range maxWorkers {
		wg.Add(1)
		go cg.processWorkerJobs(jobs, results, baseFile, &wg)
	}

	for i, row := range cg.csvData {
		workerID := fmt.Sprintf("worker-%d", i)
		workerTmpDir := filepath.Join(cg.TempDir(), workerID)
		if err := os.MkdirAll(workerTmpDir, 0755); err != nil {
			results <- generationResult{index: i, outputFile: "", err: fmt.Errorf("failed to create worker tmp dir: %w", err)}
			continue
		}

		jobs <- generationJob{index: i, data: row, tmpDir: workerTmpDir}
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	return cg.aggregateResults(results, len(cg.csvData))
}

func (cg *CertificateGenerator) loadCSVData() ([]map[string]string, error) {
	if cg.CSVPath == "" {
		return []map[string]string{}, nil
	}

	records, err := ReadCSVFromFile(cg.CSVPath)
	if err != nil {
		return nil, err
	}

	return ParseCSVToMap(records)
}

func DeterminWorkers(jobCount int) int {
	return min(max(runtime.GOMAXPROCS(0)*2, 1), jobCount)
}

func (cg *CertificateGenerator) processWorkerJobs(jobs <-chan generationJob, results chan<- generationResult, baseFile string, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		outputFile, certId, err := cg.generateSingleCertificateFromJob(job, baseFile)
		results <- generationResult{
			id:         certId,
			index:      job.index,
			outputFile: outputFile,
			err:        err,
		}

		// Update progress after each certificate is generated
		if err == nil {
			cg.incrementProgress()
		}

		os.RemoveAll(job.tmpDir)
	}
}

func (cg *CertificateGenerator) generateSingleCertificateFromJob(job generationJob, baseFile string) (string, string, error) {
	certId := uuid.NewString()

	workerBaseFile := filepath.Join(job.tmpDir, "base.pdf")
	if err := copyFile(baseFile, workerBaseFile); err != nil {
		return "", certId, err
	}

	currentFile := workerBaseFile

	for page, colAnnots := range cg.Annotations.PageColumnAnnotations {
		for _, annot := range colAnnots {
			modifiedAnnot := annot
			modifiedAnnot.Value = job.data[annot.Value]

			var err error
			currentFile, err = cg.embedTextAnnotation(currentFile, page, modifiedAnnot, job.tmpDir)
			if err != nil {
				return "", certId, fmt.Errorf("failed to apply text annotation on page %d for row %d: %w", page, job.index, err)
			}
		}
	}

	if cg.Settings.EmbedQRCode {
		_, err := cg.embedQRCode(currentFile, certId, job.tmpDir, job.index)
		if err != nil {
			return "", certId, err
		}
	}

	outputFile := filepath.Join(cg.OutputDir(), fmt.Sprintf(cg.OutFilePattern, fmt.Sprint(job.index+1))+".pdf")
	if err := os.Rename(currentFile, outputFile); err != nil {
		return "", certId, fmt.Errorf("failed to finalize certificate for row %d: %w", job.index, err)
	}

	return outputFile, certId, nil
}

func (cg *CertificateGenerator) embedQRCode(currentFile, certId, tmpDir string, index int) (string, error) {
	tmpQrCodeFile, err := os.CreateTemp(tmpDir, "autocert_qr_*.pdf")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpQrCodeFile.Name())

	err = GenerateQRCodeAsPdfByPdfPage(fmt.Sprintf(cg.Settings.QrURLPattern, certId), currentFile, 1, tmpQrCodeFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to generate QR code for row %d: %w", index, err)
	}

	err = EmbedQRCodeToPdf(currentFile, currentFile, tmpQrCodeFile.Name(), []string{})
	if err != nil {
		return "", fmt.Errorf("failed to embed QR code for row %d: %w", index, err)
	}

	return currentFile, nil
}

func (cg *CertificateGenerator) aggregateResults(results <-chan generationResult, totalCount int) ([]GeneratedResult, error) {
	resultMap := make(map[int]generationResult)
	inFile := make([]string, totalCount)
	var firstErr error

	for r := range results {
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
		} else {
			resultMap[r.index] = r
		}
	}

	if firstErr != nil {
		return nil, firstErr
	}

	generatedFiles := make([]GeneratedResult, 0, totalCount)
	for i := range totalCount {
		if r, ok := resultMap[i]; ok {
			generatedFiles = append(generatedFiles, GeneratedResult{
				Number:   i + 1,
				FilePath: r.outputFile,
				FileName: filepath.Base(r.outputFile),
				Type:     CertificateTypeNormal,
				ID:       r.id,
			})
			inFile[i] = r.outputFile
		} else {
			return nil, fmt.Errorf("missing result for row %d", i)
		}
	}

	if cg.Settings.ZipAfterGenerate {
		zipNow := time.Now()

		cg.updateProgress("Creating ZIP archive")
		zipOut := filepath.Join(cg.OutputDir(), "certificates.zip")
		err := ZipFiles(inFile, zipOut)
		if err != nil {
			return nil, fmt.Errorf("failed to zip generated files: %w", err)
		}
		generatedFiles = append(generatedFiles, GeneratedResult{
			Number:   -2,
			FilePath: zipOut,
			FileName: filepath.Base(zipOut),
			Type:     CertificateTypeZip,
			ID:       uuid.NewString(),
		})

		cg.updateProgress(fmt.Sprintf("ZIP archive created in %s", time.Since(zipNow).Truncate(time.Second)))
	}

	if cg.Settings.MergeAfterGenerate {
		mergeNow := time.Now()

		cg.updateProgress("Merging PDF files")
		mergeOut := filepath.Join(cg.OutputDir(), fmt.Sprintf(cg.OutFilePattern, "merged")+".pdf")
		err := MergePdf(inFile, mergeOut)
		if err != nil {
			return nil, fmt.Errorf("failed to merge PDF files: %w", err)
		}
		generatedFiles = append(generatedFiles, GeneratedResult{
			Number:   -1,
			FilePath: mergeOut,
			FileName: filepath.Base(mergeOut),
			Type:     CertificateTypeMerged,
			ID:       uuid.NewString(),
		})

		cg.updateProgress(fmt.Sprintf("PDF files merged in %s", time.Since(mergeNow).Truncate(time.Second)))
	}

	cg.updateProgress("Generation complete")

	return generatedFiles, nil
}
