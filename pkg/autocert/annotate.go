package autocert

type AnnotateType string

const (
	AnnotateTypeColumn    AnnotateType = "column"
	AnnotateTypeSignature AnnotateType = "signature"
)

type Position struct {
	X float64 `json:"x" form:"x" binding:"required"`
	Y float64 `json:"y" form:"y" binding:"required"`
}

type Size struct {
	Width  float64 `json:"width" form:"width" binding:"required"`
	Height float64 `json:"height" form:"height" binding:"required"`
}

type BaseAnnotate struct {
	ID       string       `json:"id" form:"id" binding:"required"`
	Type     AnnotateType `json:"type" form:"type" binding:"required"`
	Position `json:"position" form:"position" binding:"required"`
	Size     `json:"size" form:"size" binding:"required"`
}

type ColumnAnnotate struct {
	BaseAnnotate
	// column name in the CSV file
	Value          string     `json:"value" form:"value" binding:"required"`
	FontName       string     `json:"fontName" form:"fontName"`
	FontColor      string     `json:"fontColor" form:"fontColor"`
	FontSize       float64    `json:"fontSize" form:"fontSize"`
	FontWeight     FontWeight `json:"fontWeight" form:"fontWeight"`
	TextFitRectBox bool       `json:"textFitRectBox" form:"textFitRectBox"`
	TextAlign      TextAlign  `json:"textAlign" form:"textAlign"`
}

func (ca ColumnAnnotate) Font() *Font {
	return &Font{
		Name:   ca.FontName,
		Color:  ca.FontColor,
		Size:   ca.FontSize,
		Weight: ca.FontWeight,
	}
}

func (ca ColumnAnnotate) Rect() *Rect {
	return &Rect{
		Width:  ca.Size.Width,
		Height: ca.Size.Height,
	}
}

type SignatureAnnotate struct {
	BaseAnnotate
	SignatureFilePath string `json:"signatureFilePath"`
	Email             string `json:"email" form:"email" binding:"required"`
}

func (sa SignatureAnnotate) Rect() *Rect {
	return &Rect{
		Width:  sa.Size.Width,
		Height: sa.Size.Height,
	}
}

// Each page has a list of annotates
type PageSignatureAnnotations map[uint][]SignatureAnnotate
type PageColumnAnnotations map[uint][]ColumnAnnotate
type PageAnnotations struct {
	PageSignatureAnnotations PageSignatureAnnotations
	PageColumnAnnotations    PageColumnAnnotations
}
