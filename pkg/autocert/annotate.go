package autocert

type AnnotateType string

const (
	AnnotateTypeColumn    AnnotateType = "column"
	AnnotateTypeSignature AnnotateType = "signature"
)

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Size struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type BaseAnnotate struct {
	ID   string       `json:"id"`
	Type AnnotateType `json:"type"`
	Position
	Size
}

type ColumnAnnotate struct {
	BaseAnnotate
	// column name in the CSV file
	Value      string     `json:"value"`
	FontName   string     `json:"fontName"`
	FontColor  string     `json:"fontColor"`
	FontSize   float64    `json:"fontSize"`
	FontWeight FontWeight `json:"fontWeight"`
}

type SignatureAnnotate struct {
	BaseAnnotate
	SignatureFilePath string `json:"signatureFilePath"`
	Email             string `json:"email"`
}

// Each page has a list of annotates
type PageSignatureAnnotations map[uint][]SignatureAnnotate
type PageColumnAnnotations map[uint][]ColumnAnnotate
type PageAnnotations struct {
	PageSignatureAnnotations PageSignatureAnnotations
	PageColumnAnnotations    PageColumnAnnotations
}
