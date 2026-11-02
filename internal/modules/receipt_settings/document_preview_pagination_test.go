package receipt_settings

import (
	"context"
	"fmt"
	"html"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/chromedp/cdproto/page"
	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"pos-backend/internal/platform/dochtml"
	"pos-backend/internal/platform/htmlpdf"
)

// Check the actual Paged.js output, not merely source HTML or CSS declarations.
func TestDocumentPreviewPagedJSRowsAndTotals(t *testing.T) {
	for _, variant := range []struct {
		key    string
		render func(dochtml.DocData, dochtml.StoreInfo, int) (string, error)
	}{{"standard", dochtml.RenderUnifiedDocumentCopies}, {"classic", dochtml.RenderReferenceDocumentCopies}, {"template3", dochtml.RenderTemplate3DocumentCopies}} {
		for _, count := range []int{2, 15, 38} {
			t.Run(fmt.Sprintf("%s-%d", variant.key, count), func(t *testing.T) {
				doc := documentPreviewFixture()
				doc.Items = doc.Items[:count]
				html, err := variant.render(doc, dochtml.StoreInfo{Name: "Preview Store"}, 0)
				if err != nil {
					t.Fatal(err)
				}
				opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
				if path := os.Getenv("CHROME_PATH"); path != "" {
					opts = append(opts, chromedp.ExecPath(path))
				} else if runtime.GOOS == "darwin" {
					for _, path := range []string{"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser", "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"} {
						if _, err := os.Stat(path); err == nil {
							opts = append(opts, chromedp.ExecPath(path))
							break
						}
					}
				}
				alloc, stopAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
				defer stopAlloc()
				ctx, stopBrowser := chromedp.NewContext(alloc)
				defer stopBrowser()
				ctx, stop := context.WithTimeout(ctx, 75*time.Second)
				defer stop()
				var result struct {
					Pages           int   `json:"pages"`
					Rows            int   `json:"rows"`
					Headers         int   `json:"headers"`
					Numbered        int   `json:"numbered"`
					FinalTotal      bool  `json:"finalTotal"`
					EndingInside    bool  `json:"endingInside"`
					LastSignature   bool  `json:"lastSignature"`
					RowsInside      bool  `json:"rowsInside"`
					Sequential      bool  `json:"sequential"`
					HeadersOnData   bool  `json:"headersOnData"`
					NoBlankPages    bool  `json:"noBlankPages"`
					HeaderOverlap   bool  `json:"headerOverlap"`
					HeaderFullWidth bool  `json:"headerFullWidth"`
					PerPage         []int `json:"perPage"`
				}
				err = chromedp.Run(ctx, chromedp.Navigate("about:blank"), chromedp.ActionFunc(func(ctx context.Context) error {
					tree, err := page.GetFrameTree().Do(ctx)
					if err != nil {
						return err
					}
					return page.SetDocumentContent(tree.Frame.ID, htmlpdf.PreparePreview(html)).Do(ctx)
				}), chromedp.Evaluate(`window.__posPagedReady.then(()=>{const pages=[...document.querySelectorAll('.pagedjs_page')]; const selector='table.items tbody tr:not(.filler),table.t3-items tbody tr:not(.filler)'; const grouped=pages.map(p=>[...p.querySelectorAll(selector)]);const rows=grouped.map(list=>list.length);let last=0;const end=pages.at(-1)?.querySelector('.summary,.ref-total,.t3-summary'),endRect=end?.getBoundingClientRect(),pageRect=pages.at(-1)?.getBoundingClientRect();return {pages:pages.length,rows:rows.reduce((a,b)=>a+b,0),perPage:rows,headers:pages.filter(p=>p.querySelector('table.items thead th,table.t3-items thead th')).length,headersOnData:grouped.every((list,i)=>!list.length||!!pages[i].querySelector('table.items thead th,table.t3-items thead th')),noBlankPages:grouped.every((list,i)=>list.length>0||!!pages[i].querySelector('.summary,.ref-total,.t3-summary')),headerFullWidth:pages.filter(p=>p.querySelector('.pos-continuation-header')).every(p=>{const h=p.querySelector('.pos-continuation-header').getBoundingClientRect(),margin=p.querySelector('.pagedjs_margin-top').getBoundingClientRect();return Math.abs(h.left-margin.left)<2&&Math.abs(h.right-margin.right)<2}),headerOverlap:pages.some(p=>{const a=p.querySelector('.pos-continuation-header')?.getBoundingClientRect();return !!a&&[p.querySelector('.t3-items thead tr,.t3-bill thead tr'),p.querySelector('.t3-items tbody tr,.t3-bill tbody tr'),p.querySelector('.pos-page-number')].some(node=>{const b=node?.getBoundingClientRect();return !!b&&a.top<b.bottom&&a.bottom>b.top&&a.left<b.right&&a.right>b.left})}),numbered:pages.filter(p=>p.querySelector('.pos-page-number')).length,finalTotal:!!end,endingInside:!!endRect&&!!pageRect&&endRect.top>=pageRect.top-1&&endRect.bottom<=pageRect.bottom+1,lastSignature:!!pages.at(-1)?.querySelector('.pos-page-signatures,.ref-sign'),rowsInside:grouped.every((list,i)=>list.every(row=>{const r=row.getBoundingClientRect(),p=pages[i].getBoundingClientRect();return r.top>=p.top-1&&r.bottom<=p.bottom+1})),sequential:grouped.flat().every(row=>Number(row.querySelector('td')?.textContent?.trim())===++last)};})`, &result, func(p *cdpruntime.EvaluateParams) *cdpruntime.EvaluateParams { return p.WithAwaitPromise(true) }))
				if err != nil {
					t.Fatal(err)
				}
				t.Logf("%s %d: %+v", variant.key, count, result)
				if result.Rows != count || result.Numbered != result.Pages || !result.FinalTotal || !result.EndingInside || !result.LastSignature || !result.RowsInside || !result.Sequential || !result.HeadersOnData || !result.NoBlankPages || result.HeaderOverlap || (variant.key == "template3" && !result.HeaderFullWidth) || (count == 38 && result.Pages < 2) || (count == 2 && result.Pages != 1) {
					t.Fatalf("invalid pagination: %+v", result)
				}
				for i, rows := range result.PerPage {
					if rows > 0 && result.Headers <= i {
						t.Fatalf("page %d missing table header: %+v", i+1, result)
					}
				}
			})
		}
	}
}

func TestDocumentPreviewPagedJSInsideSandboxedIframe(t *testing.T) {
	doc := documentPreviewFixture()
	raw, err := dochtml.RenderUnifiedDocumentCopies(doc, dochtml.StoreInfo{Name: "Preview Store"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	iframeHTML := `<html><body><iframe sandbox="allow-same-origin allow-scripts" srcdoc="` + html.EscapeString(htmlpdf.PreparePreview(raw)) + `"></iframe></body></html>`
	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	if path := os.Getenv("CHROME_PATH"); path != "" {
		opts = append(opts, chromedp.ExecPath(path))
	} else if runtime.GOOS == "darwin" {
		for _, path := range []string{"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser", "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"} {
			if _, err := os.Stat(path); err == nil {
				opts = append(opts, chromedp.ExecPath(path))
				break
			}
		}
	}
	alloc, stopAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer stopAlloc()
	ctx, stopBrowser := chromedp.NewContext(alloc)
	defer stopBrowser()
	ctx, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	var result int
	err = chromedp.Run(ctx, chromedp.Navigate("about:blank"), chromedp.ActionFunc(func(ctx context.Context) error {
		tree, err := page.GetFrameTree().Do(ctx)
		if err != nil {
			return err
		}
		return page.SetDocumentContent(tree.Frame.ID, iframeHTML).Do(ctx)
	}), chromedp.Evaluate(`new Promise((resolve,reject)=>{const frame=document.querySelector('iframe');if(frame.contentWindow.__posPagedReady){frame.contentWindow.__posPagedReady.then(()=>resolve(frame.contentDocument.querySelectorAll('.pagedjs_page').length),reject);return;}frame.addEventListener('load',()=>{if(!frame.contentWindow.__posPagedReady){reject('missing ready promise');return;}frame.contentWindow.__posPagedReady.then(()=>resolve(frame.contentDocument.querySelectorAll('.pagedjs_page').length),reject)},{once:true})})`, &result, func(p *cdpruntime.EvaluateParams) *cdpruntime.EvaluateParams { return p.WithAwaitPromise(true) }))
	if err != nil {
		t.Fatalf("sandboxed preview failed: %v", err)
	}
	if result < 2 {
		t.Fatalf("38 items must span pages in sandboxed iframe, got %d", result)
	}
}
