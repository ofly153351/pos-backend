package htmlpdf

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

func TestRender_MultiPage(t *testing.T) {
	// Two A4 sheets separated by a forced page break.
	html := `<!DOCTYPE html><html><head><meta charset="utf-8">
<style>@page{size:A4;margin:0} .s{height:297mm;background:#eee}</style></head>
<body><div class="s">หน้า 1 ต้นฉบับ (Original)</div>
<div class="s" style="page-break-before:always">หน้า 2 สำเนา (Copy)</div></body></html>`

	pdf, err := Render(context.Background(), html)
	if err != nil {
		t.Fatalf("render (Chrome installed?): %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("not a PDF: %q", pdf[:min(8, len(pdf))])
	}
	// A two-sheet document must span multiple pages (exact count varies by Chrome's
	// PDF object layout, so assert >= 2 rather than an exact number).
	pages := bytes.Count(pdf, []byte("/Type /Page")) - bytes.Count(pdf, []byte("/Type /Pages"))
	if pages < 2 {
		t.Fatalf("want >= 2 pages, got %d", pages)
	}
	if len(pdf) < 1000 {
		t.Fatalf("PDF suspiciously small: %d bytes", len(pdf))
	}
}

func TestWithPagedJSAddsPageNumbersAfterRendering(t *testing.T) {
	html := withPagedJS("<!doctype html><html><head></head><body></body></html>")
	for _, marker := range []string{
		"class PosCopyPageNumbers extends Paged.Handler",
		"afterRendered(pages)",
		"page.element || page",
		"label.className = \"pos-page-number\"",
		"label.textContent = posCopyPage + \"/\" + copyPages.length",
		"class PosBottomSignatures extends Paged.Handler",
		".signature-source[data-copy-purpose]",
		"pageElement.appendChild(clone)",
	} {
		if !strings.Contains(html, marker) {
			t.Fatalf("page-number handler is missing %q", marker)
		}
	}
	preview := PreparePreview("<!doctype html><html><head></head><body></body></html>")
	if !strings.Contains(preview, "window.__posPagedReady=") || !strings.Contains(preview, "return window.PagedPolyfill.preview();") {
		t.Fatal("preview HTML must start Paged.js pagination")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type signaturePosition struct {
	Copy       string  `json:"copy"`
	OnLast     bool    `json:"onLast"`
	Top        float64 `json:"top"`
	Bottom     float64 `json:"bottom"`
	PageEnd    float64 `json:"pageEnd"`
	AreaBottom float64 `json:"areaBottom"`
}

func TestPagedSignatureHandlerPinsEachCopyToItsLastPage(t *testing.T) {
	copyHTML := func(copy string) string {
		body := strings.Repeat(`<div class="line">A flow fragment for `+copy+`.</div>`, 50)
		return `<div class="copy-break"><div class="page" data-copy-purpose="` + copy + `" data-show-signature="true">` + body + `<div class="signature-source" data-copy-purpose="` + copy + `">SIGN-` + copy + `</div></div></div>`
	}
	html := `<!doctype html><html lang="th"><head><meta charset="utf-8"><style>
@page signed-document{size:A4;margin:20mm 12mm 47mm;}
@page signed-document:first{margin-top:12mm;}
.page{page:signed-document;}
.pagedjs_page{position:relative;}
.copy-break{break-after:page;page-break-after:always;}
.signature-source{display:none!important;}
.pos-page-signatures{position:absolute;left:12mm;right:12mm;bottom:16mm;margin:0;}
.line{height:10mm;break-inside:avoid;}
</style></head><body>` + copyHTML("Original") + copyHTML("Copy") + `</body></html>`
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", true), chromedp.Flag("no-sandbox", true), chromedp.Flag("disable-gpu", true))
	if path := resolveExecPath(); path != "" {
		opts = append(opts, chromedp.ExecPath(path))
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()
	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()
	ctx, cancelTimeout := context.WithTimeout(ctx, 60*time.Second)
	defer cancelTimeout()

	var result struct {
		PageCount  int                 `json:"pageCount"`
		Signatures []signaturePosition `json:"signatures"`
	}
	err := chromedp.Run(ctx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			tree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(tree.Frame.ID, withPagedJS(html)).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var ready bool
			return chromedp.Evaluate(`(async()=>{await PagedPolyfill.preview();return true})()`, &ready, func(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) }).Do(ctx)
		}),
		chromedp.Evaluate(`(()=>{const pages=[...document.querySelectorAll(".pagedjs_page")];const signatures=[...document.querySelectorAll(".pos-page-signatures")].map(node=>{const copy=node.getAttribute("data-copy-purpose")||"";const matching=pages.filter(page=>{const marker=page.querySelector("[data-copy-purpose]");return (marker?.getAttribute("data-copy-purpose")||"")===copy});const page=node.closest(".pagedjs_page");const r=node.getBoundingClientRect();const p=page.getBoundingClientRect();const a=page.querySelector(".pagedjs_area").getBoundingClientRect();return {copy,onLast:page===matching.at(-1),top:r.top,bottom:r.bottom,pageEnd:p.bottom,areaBottom:a.bottom}});return {pageCount:pages.length,signatures}})()`, &result),
	)
	if err != nil {
		t.Fatalf("run Paged.js signature layout: %v", err)
	}
	if result.PageCount < 4 {
		t.Fatalf("expected multiple pages for two copies, got %d", result.PageCount)
	}
	if len(result.Signatures) != 2 {
		t.Fatalf("expected one pinned signature per copy, got %+v", result.Signatures)
	}
	for _, signature := range result.Signatures {
		if !signature.OnLast {
			t.Errorf("%s signature is not on that copy's last page", signature.Copy)
		}
		if signature.Top <= signature.AreaBottom {
			t.Errorf("%s signature overlaps document content area: top %.1fpx, content bottom %.1fpx", signature.Copy, signature.Top, signature.AreaBottom)
		}
		if delta := signature.PageEnd - signature.Bottom; delta < 58 || delta > 63 {
			t.Errorf("%s signature bottom offset = %.1fpx, want approximately 16mm", signature.Copy, delta)
		}
	}
}
