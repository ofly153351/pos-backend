package dochtml

import (
	"bytes"
	"context"
	"os"
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"

	"pos-backend/internal/platform/htmlpdf"
)

func template3Fixture(kind string, count int) DocData {
	items := make([]DocItem, count)
	for i := range items {
		items[i] = DocItem{Description: "REAL ITEM", Unit: "ชิ้น", Quantity: 1, UnitPrice: 100, Amount: 100}
	}
	return DocData{Type: kind, DocumentNoFull: "T3-" + kind, DocumentDate: time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC), CustomerName: "REAL BUYER", Items: items, BillRows: []BillRow{{DocumentNo: "DO-REAL", Amount: 100}}, Subtotal: float64(count) * 100, VatRate: 7, VatAmount: float64(count) * 7, TotalAmount: float64(count) * 107}
}

func TestTemplate3RendersEveryA4DocumentAndCopies(t *testing.T) {
	for _, kind := range []string{"QUOTATION", "INVOICE", "RECEIPT", "TAX_INVOICE", "BILL", "CREDIT_NOTE", "DELIVERY_ORDER"} {
		t.Run(kind, func(t *testing.T) {
			html, err := RenderTemplate3DocumentCopies(template3Fixture(kind, 1), StoreInfo{Name: "REAL STORE", Address: "REAL ADDRESS", LogoURL: "data:image/png;base64,aGVsbG8="}, -1)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"T3-" + kind, "REAL STORE", "REAL ADDRESS", "REAL BUYER", "class=\"t3-header\"", "class=\"t3-title\"", "class=\"t3-signatures signature-source\"", "data:image/png;base64,aGVsbG8="} {
				if !strings.Contains(html, want) {
					t.Errorf("missing %q", want)
				}
			}
			if kind == "BILL" {
				if !strings.Contains(html, "DO-REAL") {
					t.Error("billing register missing")
				}
			} else if !strings.Contains(html, "REAL ITEM") {
				t.Error("product missing")
			}
			if kind != "QUOTATION" && strings.Contains(html, `class="t3-approval-box"`) {
				t.Error("quotation approval on other document")
			}
			if strings.Contains(html, "JO187/2026") || strings.Contains(html, "JOSEPH KORAT") {
				t.Error("sample's business data leaked")
			}
		})
	}
}

func TestTemplate3HeaderKeepsReferenceProportions(t *testing.T) {
	html, err := RenderTemplate3DocumentCopies(template3Fixture("QUOTATION", 1), StoreInfo{Name: "Store"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `.t3-header{display:flex;justify-content:space-between;align-items:stretch;gap:8mm;min-height:68mm;`) || !strings.Contains(html, `.t3-store{align-self:flex-start;`) || !strings.Contains(html, `.t3-title{align-self:flex-end;`) {
		t.Fatal("template 3 title must sit below store identity before the item table")
	}
}

func TestTemplate3RichSingleItemPDF(t *testing.T) {
	taxID := "0123456789012"
	note := "กรุณาตรวจสอบสินค้าและจำนวนก่อนรับมอบ ติดต่อร้านหากพบความผิดปกติ"
	date := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	doc := template3Fixture("QUOTATION", 1)
	doc.CustomerName = "บริษัท ตัวอย่างการค้าส่งและบริการ จำกัด (สำนักงานใหญ่)"
	doc.CustomerAddress = "99/11 ถนนสุขุมวิท แขวงคลองตันเหนือ เขตวัฒนา กรุงเทพมหานคร 10110"
	doc.CustomerTaxID = &taxID
	doc.CustomerPhone = "02-123-4567"
	doc.DeliveryAddress = "88/9 ถนนพระราม 4 แขวงคลองตัน เขตคลองเตย กรุงเทพมหานคร 10110"
	doc.Notes = &note
	doc.DocumentDate = date
	doc.Items[0].Description = "สินค้าตัวอย่างสำหรับจัดส่งหนึ่งรายการ พร้อมรายละเอียดเพิ่มเติม"
	html, err := RenderTemplate3DocumentCopies(doc, StoreInfo{Name: "บริษัท ร้านค้าตัวอย่างและจัดจำหน่ายสินค้า จำกัด", Address: "123 ถนนสุขุมวิท แขวงคลองตันเหนือ เขตวัฒนา กรุงเทพมหานคร 10110", Phone: "02-999-9999", TaxID: taxID}, 0)
	if err != nil {
		t.Fatal(err)
	}
	pdf, err := htmlpdf.Render(context.Background(), html)
	if err != nil {
		t.Fatal(err)
	}
	if pages := bytes.Count(pdf, []byte("/Type /Page")) - bytes.Count(pdf, []byte("/Type /Pages")); pages != 1 {
		t.Fatalf("rich one-item document uses %d pages, want 1", pages)
	}
}

func TestTemplate3OnlyAddsDiscountColumnWhenNeeded(t *testing.T) {
	d := template3Fixture("QUOTATION", 1)
	html, err := RenderTemplate3DocumentCopies(d, StoreInfo{Name: "Store"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "<th>ส่วนลด</th>") {
		t.Fatal("zero-discount quotation must match six-column sample")
	}
	d.Items[0].DiscountValue = 10
	html, err = RenderTemplate3DocumentCopies(d, StoreInfo{Name: "Store"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<th>ส่วนลด</th>") {
		t.Fatal("real discount must remain visible")
	}
}

func TestTemplate3ShortPDFAllA4Types(t *testing.T) {
	for _, kind := range []string{"QUOTATION", "INVOICE", "RECEIPT", "TAX_INVOICE", "BILL", "CREDIT_NOTE", "DELIVERY_ORDER"} {
		t.Run(kind, func(t *testing.T) {
			html, err := RenderTemplate3DocumentCopies(template3Fixture(kind, 1), StoreInfo{Name: "Store"}, 0)
			if err != nil {
				t.Fatal(err)
			}
			pdf, err := htmlpdf.Render(context.Background(), html)
			if err != nil {
				t.Fatal(err)
			}
			pages := bytes.Count(pdf, []byte("/Type /Page")) - bytes.Count(pdf, []byte("/Type /Pages"))
			if pages != 1 {
				t.Errorf("short %s PDF has %d pages, want 1", kind, pages)
			}
		})
	}
}

func TestTemplate3LongPrintPreviewKeepsRowsAndLastPageSignature(t *testing.T) {
	html, err := RenderTemplate3DocumentCopies(template3Fixture("QUOTATION", 60), StoreInfo{Name: "Store"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	if chrome := os.Getenv("CHROME_PATH"); chrome != "" {
		opts = append(opts, chromedp.ExecPath(chrome))
	} else if goruntime.GOOS == "darwin" {
		for _, chrome := range []string{"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser", "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge", "/Applications/Chromium.app/Contents/MacOS/Chromium"} {
			if info, err := os.Stat(chrome); err == nil && !info.IsDir() {
				opts = append(opts, chromedp.ExecPath(chrome))
				break
			}
		}
	}
	alloc, stopAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer stopAlloc()
	ctx, stopBrowser := chromedp.NewContext(alloc)
	defer stopBrowser()
	ctx, stopTimeout := context.WithTimeout(ctx, 60*time.Second)
	defer stopTimeout()
	var result struct {
		Pages      int  `json:"pages"`
		Rows       int  `json:"rows"`
		Signatures int  `json:"signatures"`
		LastPage   bool `json:"lastPage"`
	}
	err = chromedp.Run(ctx, chromedp.Navigate("about:blank"), chromedp.ActionFunc(func(ctx context.Context) error {
		tree, err := page.GetFrameTree().Do(ctx)
		if err != nil {
			return err
		}
		return page.SetDocumentContent(tree.Frame.ID, htmlpdf.PreparePreview(html)).Do(ctx)
	}), chromedp.Evaluate(`window.__posPagedReady.then(()=>{const p=[...document.querySelectorAll('.pagedjs_page')],s=[...document.querySelectorAll('.pos-page-signatures')];return {pages:p.length,rows:document.querySelectorAll('.pagedjs_page .t3-items tbody tr').length,signatures:s.length,lastPage:s.length===1&&s[0].closest('.pagedjs_page')===p[p.length-1]}})`, &result, func(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) }))
	if err != nil {
		t.Fatalf("Template 3 Paged.js preview: %v", err)
	}
	if result.Pages < 2 || result.Rows != 60 || result.Signatures != 1 || !result.LastPage {
		t.Fatalf("rows/signature pagination: %+v", result)
	}
}

func TestTemplate3ShortPDFAndLongPagination(t *testing.T) {
	for _, count := range []int{1, 60} {
		html, err := RenderTemplate3DocumentCopies(template3Fixture("QUOTATION", count), StoreInfo{Name: "Store", Address: "123 Example Road", LogoURL: ""}, 0)
		if err != nil {
			t.Fatal(err)
		}
		if count == 1 {
			if path := os.Getenv("TEMPLATE3_RAW_HTML_OUT"); path != "" {
				if err := os.WriteFile(path, []byte(html), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if path := os.Getenv("TEMPLATE3_PREVIEW_HTML_OUT"); path != "" {
				if err := os.WriteFile(path, []byte(htmlpdf.PreparePreview(html)), 0o600); err != nil {
					t.Fatal(err)
				}
			}
		} else if path := os.Getenv("TEMPLATE3_LONG_PREVIEW_OUT"); path != "" {
			if err := os.WriteFile(path, []byte(htmlpdf.PreparePreview(html)), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		pdf, err := htmlpdf.Render(context.Background(), html)
		if err != nil {
			t.Fatal(err)
		}
		if count == 1 {
			if path := os.Getenv("TEMPLATE3_PDF_OUT"); path != "" {
				if err := os.WriteFile(path, pdf, 0o600); err != nil {
					t.Fatal(err)
				}
			}
		}
		pages := bytes.Count(pdf, []byte("/Type /Page")) - bytes.Count(pdf, []byte("/Type /Pages"))
		if count == 1 && pages != 1 {
			t.Errorf("one item: got %d PDF pages, want 1", pages)
		}
		if count == 60 && pages < 2 {
			t.Errorf("60 items: got %d PDF pages, want multiple", pages)
		}
	}
}
