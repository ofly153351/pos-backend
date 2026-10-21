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
	_ "embed"
	"fmt"
	"os"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

// Paged.js 0.4.3 is vendored so PDF generation is deterministic and does not
// depend on a CDN or network access at render time.
//
//go:embed assets/paged.polyfill.min.js
var pagedPolyfill []byte

func PreparePreview(html string) string {
	prepared := withPagedJS(html)
	previewScript := `<script>window.__posPagedReady=(async()=>{const imgs=Array.from(document.images);await Promise.all(imgs.map((img)=>img.complete?Promise.resolve():new Promise((resolve)=>{img.addEventListener("load",resolve,{once:true});img.addEventListener("error",resolve,{once:true});})));if(document.fonts?.ready)await document.fonts.ready;return window.PagedPolyfill.preview();})();</script>`
	if strings.Contains(prepared, "</body>") {
		return strings.Replace(prepared, "</body>", previewScript+"</body>", 1)
	}
	return prepared + previewScript
}

func withPagedJS(html string) string {
	config := `<script>window.PagedConfig={auto:false};</script>`
	script := `<script>` + string(pagedPolyfill) + `</script>`
	repeatHeaders := `<script>
class PosRepeatTableHeaders extends Paged.Handler {
  afterPageLayout(pageElement) {
    pageElement.querySelectorAll("table[data-split-from]").forEach((table) => {
      if (table.firstElementChild?.tagName === "THEAD") return;
      const ref = table.dataset.ref;
      const source = ref && this.chunker?.source?.querySelector("[data-ref='" + ref + "']");
      const header = source?.querySelector("thead");
      if (header) table.insertBefore(header.cloneNode(true), table.firstChild);
    });
  }
}
Paged.registerHandlers(PosRepeatTableHeaders);
class PosContinuationHeaders extends Paged.Handler {
  afterRendered(pages) {
    const source = document.querySelector(".hdr-mini");
    if (!source) return;
    const firstPageByCopy = new Set();
    pages.forEach((page) => {
      const pageElement = page.element || page;
      const marker = pageElement.querySelector("[data-copy-purpose]");
      const copy = marker?.getAttribute("data-copy-purpose") || "";
      if (!firstPageByCopy.has(copy)) {
        firstPageByCopy.add(copy);
        return;
      }
      if (pageElement.querySelector(".pos-continuation-header")) return;
      const header = source.cloneNode(true);
      header.className = "pos-continuation-header";
      pageElement.appendChild(header);
    });
  }
}
Paged.registerHandlers(PosContinuationHeaders);
let posActiveCopy = "";
let posCopyPage = 0;
class PosCopyPageNumbers extends Paged.Handler {
  afterRendered(pages) {
    document.querySelectorAll(".screen-page-number").forEach((node) => node.remove());
    pages.forEach((page) => {
      const pageElement = page.element || page;
      if (pageElement.querySelector(".pos-page-number")) return;
      const marker = pageElement.querySelector("[data-copy-purpose]");
      const copy = marker?.getAttribute("data-copy-purpose") || "";
      if (copy && copy !== posActiveCopy) {
        posActiveCopy = copy;
        posCopyPage = 0;
      }
      posCopyPage += 1;
      const copyPages = pages.filter((candidate) => {
        const candidateElement = candidate.element || candidate;
        const candidateMarker = candidateElement.querySelector("[data-copy-purpose]");
        return (candidateMarker?.getAttribute("data-copy-purpose") || "") === copy;
      });
      const label = document.createElement("div");
      label.className = "pos-page-number";
      label.textContent = posCopyPage + "/" + copyPages.length;
      pageElement.appendChild(label);
    });
  }
}
Paged.registerHandlers(PosCopyPageNumbers);
</script>`
	if strings.Contains(html, "</head>") {
		return strings.Replace(html, "</head>", config+script+repeatHeaders+"</head>", 1)
	}
	return config + script + repeatHeaders + html
}

func resolveExecPath() string {
	if p := os.Getenv("CHROME_PATH"); p != "" {
		return p
	}
	if goruntime.GOOS == "darwin" {
		for _, p := range []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		} {
			if info, err := os.Stat(p); err == nil && !info.IsDir() {
				return p
			}
		}
	}
	return ""
}

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
	if p := resolveExecPath(); p != "" {
		opts = append(opts, chromedp.ExecPath(p))
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(parent, opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	ctx, cancelTimeout := context.WithTimeout(ctx, 60*time.Second)
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
			return page.SetDocumentContent(tree.Frame.ID, withPagedJS(html)).Do(ctx)
		}),
		// Wait for all images (store logo, PromptPay QR) to finish loading. They are
		// fetched over the network (e.g. RustFS), so printing immediately would drop
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
		// Paged.js owns fragmentation. It repeats table headers, honors break-* rules,
		// and creates physical pages from the natural content height.
		chromedp.ActionFunc(func(ctx context.Context) error {
			var ready bool
			return chromedp.Evaluate(`(async () => {
				if (!window.PagedPolyfill) throw new Error("Paged.js failed to load");
				await window.PagedPolyfill.preview();
				return true;
			})()`, &ready, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
				return p.WithAwaitPromise(true)
			}).Do(ctx)
		}),
		// Give Paged.js and web fonts a moment to settle before printing.
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
