// Package htmlpdf converts a rendered HTML document into a PDF using headless
// Chrome (chromedp). This guarantees the downloaded PDF is pixel-identical to the
// on-screen preview / browser print, because both render the SAME HTML — unlike
// the hand-drawn gofpdf renderer, which inevitably drifts from the HTML layout.
//
// Requirement: a Chrome/Chromium/Edge binary must be available on the host.
// chromedp auto-detects the system browser. If none is found, Render returns an
// error and the caller should surface a clear message.
package htmlpdf

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// Render converts a full HTML document to PDF bytes (A4, CSS @page respected,
// backgrounds printed). A fresh headless browser is spun up per call and torn
// down — fine for the low volume of document downloads in a POS.
func Render(parent context.Context, html string) ([]byte, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
	)
	// Prod override: point at a specific Chrome/Chromium/Edge binary when the host
	// keeps it outside the default search paths (set CHROME_PATH in the env).
	if p := os.Getenv("CHROME_PATH"); p != "" {
		opts = append(opts, chromedp.ExecPath(p))
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(parent, opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	ctx, cancelTimeout := context.WithTimeout(ctx, 30*time.Second)
	defer cancelTimeout()

	var pdf []byte
	err := chromedp.Run(ctx,
		// about:blank gives us a frame to inject the document into.
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			tree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(tree.Frame.ID, html).Do(ctx)
		}),
		// Wait for all images (store logo, PromptPay QR) to finish loading. They are
		// fetched over the network (e.g. MinIO), so printing immediately would drop
		// them — the logo/QR would be missing from the PDF. Resolves on load OR error,
		// with a 5s safety cap so an unreachable asset can never hang the render.
		chromedp.ActionFunc(func(ctx context.Context) error {
			var ready bool
			return chromedp.Evaluate(`new Promise((resolve) => {
				const imgs = Array.from(document.images);
				let pending = imgs.filter((i) => !i.complete).length;
				if (pending === 0) { resolve(true); return; }
				const done = () => { if (--pending <= 0) resolve(true); };
				imgs.forEach((i) => { if (!i.complete) { i.addEventListener('load', done); i.addEventListener('error', done); } });
				setTimeout(() => resolve(true), 5000);
			})`, &ready, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
				return p.WithAwaitPromise(true)
			}).Do(ctx)
		}),
		// Give web fonts / layout a moment to settle before printing.
		chromedp.Sleep(250*time.Millisecond),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPreferCSSPageSize(true).
				Do(ctx)
			if err != nil {
				return err
			}
			pdf = buf
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("html→pdf render (is Chrome installed?): %w", err)
	}
	return pdf, nil
}
