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
	Text       string     `json:"text"`
	FontSize   float64    `json:"fontSize"`
	FontName   string     `json:"FontName"`
	FontWeight FontWeight `json:"fontWeight"`
}

type SignatureAnnotate struct {
	BaseAnnotate
	Data  string `json:"data"`
	Email string `json:"email"`
}

type AnnotateState interface {
	GetType() AnnotateType
}

func (c ColumnAnnotate) GetType() AnnotateType    { return AnnotateTypeColumn }
func (s SignatureAnnotate) GetType() AnnotateType { return AnnotateTypeSignature }
