package main

import (
	"fmt"
	"os"

	"github.com/SeakMengs/AutoCert/pkg/autocert"
)

func main() {
	cfg := autocert.NewDefaultConfig()
	tr := autocert.NewTextRenderer(cfg, autocert.Rect{Width: 200, Height: 50}, autocert.Font{
		Name:   "Microsoft YaHei",
		Size:   12,
		Color:  "#000000",
		Weight: autocert.FontWeightRegular,
	}, true, false)

	tmp1, _ := os.CreateTemp(cfg.TmpDir, "autocert-text-*.pdf")
	tmp2, _ := os.CreateTemp(cfg.TmpDir, "autocert-text-*.pdf")
	tmp3, _ := os.CreateTemp(cfg.TmpDir, "autocert-text-*.pdf")

	err := tr.RenderSvgTextAsPdf("Hello Word 与其 daskndsajk dnsajd hsanjkd sabnjkdash ndjksabnd jkas dbsajkdb nsajkd bsajkd sabnjkdsahnjkas", autocert.TextAlignCenter, tmp1.Name())
	if err != nil {
		fmt.Printf("Error rendering text: %v\n", err)
	}

	err = tr.RenderSvgTextAsPdf("Hello World 与其", autocert.TextAlignLeft, tmp2.Name())
	if err != nil {
		fmt.Printf("Error rendering text: %v\n", err)
	}

	err = tr.RenderSvgTextAsPdf("Hello World 与其", autocert.TextAlignRight, tmp3.Name())
	if err != nil {
		fmt.Printf("Error rendering text: %v\n", err)
	}

	// defer os.Remove(tmp1.Name())
	// defer os.Remove(tmp2.Name())
	// defer os.Remove(tmp3.Name())

	fmt.Printf("Text rendered successfully to %s, %s, %s\n", tmp1.Name(), tmp2.Name(), tmp3.Name())
}
