package autocert

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

/*
 * Attention: tdewolff/canvas uses mm as the unit of measurement, all input from this package take * px as the unit and will convert to mm if needed.
 */

const DPI = 72

type TextAlign int

const (
	TextAlignCenter TextAlign = iota
	TextAlignLeft
	TextAlignRight
)

// Rect accept px, will convert to mm by the system when needed
// It's the rectangle box of the text that will be rendered
type Rect struct {
	Width  float64
	Height float64
}

type TextRenderer struct {
	cfg  *Config
	rect Rect
	// Font detail such as name, size, color, weight
	font Font
	// FontFamily is the font family that will be to load font face
	fontFamily           *canvas.FontFamily
	optimizeFontSizeBool bool
	removeLineBreaksBool bool
}

// Converts pixels to millimeters
func (tr *TextRenderer) pxToMM(px float64) float64 {
	return (px * 25.4) / DPI
}

// Converts millimeters to pixels
func (tr *TextRenderer) mmToPx(mm float64) float64 {
	return (mm * DPI) / 25.4
}

func (tr *TextRenderer) getRectMM() Rect {
	return Rect{
		Width:  tr.pxToMM(tr.rect.Width),
		Height: tr.pxToMM(tr.rect.Height),
	}
}

func NewTextRenderer(cfg *Config, rect Rect, font Font, optimizeFontSize bool, removeLineBreaks bool) *TextRenderer {
	fontLoader := NewFontLoader(cfg)
	fontFamily, err := fontLoader.LoadFont(font.Name, font.GetFontStyle())
	if err != nil {
		log.Fatalf("failed to load font: %v", err)
	}
	if fontFamily == nil {
		log.Fatalf("font face is nil")
	}

	return &TextRenderer{
		cfg:                  cfg,
		rect:                 rect,
		font:                 font,
		fontFamily:           fontFamily,
		optimizeFontSizeBool: optimizeFontSize,
		removeLineBreaksBool: removeLineBreaks,
	}
}

func (tr *TextRenderer) drawText(ctx *canvas.Context, text string, alignment TextAlign) {
	fontSize := tr.font.Size
	if tr.optimizeFontSizeBool {
		fontSize, _, _ = tr.optimizeFontSize(text) // Ignoring other return values
	}

	// Create the font face
	face := tr.fontFamily.Face(fontSize, canvas.Hex(tr.font.Color), tr.font.GetFontStyle(), canvas.FontNormal)

	rt := canvas.NewRichText(face)
	rt.WriteString(text)

	rectMM := tr.getRectMM()

	textBox := rt.ToText(rectMM.Width, rectMM.Height, canvas.Left, canvas.Top, 0.0, 0.0)

	textWidthMM, textHeightMM := textBox.Bounds().W(), textBox.Bounds().H()

	centerYMM := (rectMM.Height - textHeightMM) / 2

	// Set X position based on alignment
	var xPosition float64
	switch alignment {
	case TextAlignLeft:
		xPosition = 0
	case TextAlignRight:
		xPosition = rectMM.Width - textWidthMM
	case TextAlignCenter:
		xPosition = (rectMM.Width - textWidthMM) / 2
	}

	// Draw the text
	ctx.DrawText(xPosition, centerYMM, textBox)
}

func (tr *TextRenderer) drawCenteredText(ctx *canvas.Context, text string) {
	tr.drawText(ctx, text, TextAlignCenter)
}

func (tr *TextRenderer) drawLeftAlignedText(ctx *canvas.Context, text string) {
	tr.drawText(ctx, text, TextAlignLeft)
}

func (tr *TextRenderer) drawRightAlignedText(ctx *canvas.Context, text string) {
	tr.drawText(ctx, text, TextAlignRight)
}

// OptimizeFontSize adjusts the font size to fit the text within the specified dimensions
func (tr *TextRenderer) optimizeFontSize(text string) (float64, float64, float64) {
	fontSize := tr.font.Size
	var textWidthMM, textHeightMM float64

	rectMM := tr.getRectMM()
	for {
		face := tr.fontFamily.Face(fontSize, canvas.Hex(tr.font.Color), tr.font.GetFontStyle(), canvas.FontNormal)
		textBox := canvas.NewTextBox(face, text, 0, 0, canvas.Left, canvas.Top, 0.0, 0.0)

		textWidthMM, textHeightMM = textBox.Bounds().W(), textBox.Bounds().H()

		if textWidthMM <= rectMM.Width && textHeightMM <= rectMM.Height {
			break
		}

		// log.Printf("Container width: %f, Container height: %f, Text width: %f, Text height: %f (in mm unit)", containerWidthMM, containerHeightMM, textWidth, textHeight)

		fontSize--
		if fontSize <= 0 {
			log.Printf("failed to fit text into container: font size reduced to 0")
			// Set to minimum font size
			fontSize = 0
			textWidthMM = 0
			textHeightMM = 0
			break
		}
	}

	return fontSize, textWidthMM, textHeightMM
}

func (tr *TextRenderer) removeLineBreaks(text string) string {
	re := regexp.MustCompile(`[\r\n]+`)
	return strings.TrimSpace(re.ReplaceAllString(text, ""))
}

func (tr *TextRenderer) RenderSvgTextAsPdf(text string, align TextAlign, output string) error {
	rectMM := tr.getRectMM()
	c := canvas.New(rectMM.Width, rectMM.Height)
	canvasCtx := canvas.NewContext(c)
	// Change coordination from bottom-left to top-left
	canvasCtx.SetCoordSystem(canvas.CartesianIV)

	if tr.removeLineBreaksBool {
		text = tr.removeLineBreaks(text)
	}

	switch align {
	case TextAlignCenter:
		tr.drawCenteredText(canvasCtx, text)
	case TextAlignLeft:
		tr.drawLeftAlignedText(canvasCtx, text)
	case TextAlignRight:
		tr.drawRightAlignedText(canvasCtx, text)
	default:
		tr.drawCenteredText(canvasCtx, text)
	}

	if err := renderers.Write(output, c); err != nil {
		return fmt.Errorf("failed to write PDF: %v", err)
	}

	return nil
}
