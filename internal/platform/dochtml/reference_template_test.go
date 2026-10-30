package dochtml

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"pos-backend/internal/platform/htmlpdf"
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

func TestReferenceTemplateAllDocumentsAndCopies(t *testing.T) {
	for _, kind := range []string{"QUOTATION", "INVOICE", "DELIVERY_ORDER", "RECEIPT", "TAX_INVOICE", "BILL", "CREDIT_NOTE"} {
		t.Run(kind, func(t *testing.T) {
			d := DocData{Type: kind, DocumentNoFull: "REF-" + kind, DocumentDate: time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC), CustomerName: "TEST CUSTOMER", Items: []DocItem{{Description: "TEST PRODUCT", Quantity: 2, Unit: "ชิ้น", UnitPrice: 20, Amount: 40}}, BillRows: []BillRow{{DocumentNo: "DO-REAL", Amount: 40}}, Subtotal: 40, TotalAmount: 40}
			html, err := RenderReferenceDocumentCopies(d, StoreInfo{Name: "TEST STORE"}, -1)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"REF-" + kind, "TEST STORE", "TEST CUSTOMER", "background:#b6cce3", "grid-template-columns:73% 27%", ".ref-terms", "จำนวนเงินตัวอักษร"} {
				if !strings.Contains(html, want) {
					t.Errorf("missing %q", want)
				}
			}
			if kind == "BILL" && !strings.Contains(html, "DO-REAL") {
				t.Fatal("billing register missing")
			}
			if kind != "BILL" && !strings.Contains(html, "TEST PRODUCT") {
				t.Fatal("product missing")
			}
			if strings.Contains(html, "อิน ดีไซน์") || strings.Contains(html, "นาคามรกต") || strings.Contains(html, "QU13202610-005") {
				t.Fatal("sample business data leaked")
			}
		})
	}
}

func TestReferenceTemplatePDFAllTypes(t *testing.T) {
	for _, kind := range []string{"QUOTATION", "INVOICE", "DELIVERY_ORDER", "RECEIPT", "TAX_INVOICE", "BILL", "CREDIT_NOTE"} {
		t.Run(kind, func(t *testing.T) {
			d := DocData{Type: kind, DocumentNoFull: "REF-" + kind, DocumentDate: time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC), CustomerName: "Buyer", Items: []DocItem{{Description: "Item", Quantity: 1, Unit: "ชิ้น", UnitPrice: 100, Amount: 100}}, BillRows: []BillRow{{DocumentNo: "DO-1", Amount: 100}}, Subtotal: 100, TotalAmount: 100}
			html, err := RenderReferenceDocumentCopies(d, StoreInfo{Name: "Store"}, 0)
			if err != nil {
				t.Fatal(err)
			}
			pdf, err := htmlpdf.Render(context.Background(), html)
			if err != nil {
				t.Fatal(err)
			}
			pages := bytes.Count(pdf, []byte("/Type /Page")) - bytes.Count(pdf, []byte("/Type /Pages"))
			if pages != 1 {
				t.Fatalf("short %s rendered %d pages", kind, pages)
			}
		})
	}
}

func TestReferenceTemplateRichSingleItemFitsA4(t *testing.T) {
	taxID := "0123456789012"
	notes := "กรุณาตรวจสอบสินค้าและจำนวนก่อนรับมอบ และติดต่อร้านหากพบความผิดปกติ"
	date := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	d := DocData{Type: "DELIVERY_ORDER", DocumentNoFull: "DO-20261030-0001", DocumentDate: date, DeliveryDate: &date,
		CustomerName: "บริษัท ตัวอย่างการค้าส่งและบริการ จำกัด (สำนักงานใหญ่)", CustomerAddress: "99/11 ถนนสุขุมวิท แขวงคลองตันเหนือ เขตวัฒนา กรุงเทพมหานคร 10110", CustomerPhone: "02-123-4567", CustomerTaxID: &taxID,
		DeliveryAddress: "88/9 ถนนพระราม 4 แขวงคลองตัน เขตคลองเตย กรุงเทพมหานคร 10110", DeliveryContact: "คุณผู้ประสานงาน", DeliveryPhone: "089-123-4567",
		StaffName: "พนักงานตัวอย่าง", InvoiceRefNo: "INV-20261030-0001", PORefNo: "PO-20261030-0001", Notes: &notes,
		Items: []DocItem{{Description: "สินค้าตัวอย่างสำหรับจัดส่งหนึ่งรายการ พร้อมรายละเอียดเพิ่มเติม", Quantity: 1, Unit: "ชิ้น", UnitPrice: 1000, Amount: 1000}}, Subtotal: 1000, ShippingFee: 50, PreVatAmount: 1050, VatRate: 7, VatAmount: 73.50, TotalAmount: 1123.50}
	store := StoreInfo{Name: "บริษัท ร้านค้าตัวอย่างและจัดจำหน่ายสินค้า จำกัด", Address: "123 ถนนสุขุมวิท แขวงคลองตันเหนือ เขตวัฒนา กรุงเทพมหานคร 10110", Phone: "02-999-9999", TaxID: taxID, Branch: "สำนักงานใหญ่"}
	html, err := RenderReferenceDocumentCopies(d, store, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `data-compact="true"`) {
		t.Fatal("rich document must use compact item area")
	}
	if path := os.Getenv("REFERENCE_RICH_HTML_OUT"); path != "" {
		if err := os.WriteFile(path, []byte(html), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if path := os.Getenv("REFERENCE_RICH_PREVIEW_OUT"); path != "" {
		if err := os.WriteFile(path, []byte(htmlpdf.PreparePreview(html)), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	pdf, err := htmlpdf.Render(context.Background(), html)
	if err != nil {
		t.Fatal(err)
	}
	if path := os.Getenv("REFERENCE_RICH_PDF_OUT"); path != "" {
		if err := os.WriteFile(path, pdf, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	pages := bytes.Count(pdf, []byte("/Type /Page")) - bytes.Count(pdf, []byte("/Type /Pages"))
	if pages != 1 {
		t.Fatalf("rich single-item reference document should fit on one A4 page, got %d", pages)
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
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	var preview struct {
		Pages          int  `json:"pages"`
		SectionsInside bool `json:"sectionsInside"`
	}
	err = chromedp.Run(ctx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			tree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(tree.Frame.ID, htmlpdf.PreparePreview(html)).Do(ctx)
		}),
		chromedp.Evaluate(`window.__posPagedReady.then(()=>{const page=document.querySelector('.pagedjs_page'),r=page?.getBoundingClientRect(),sections=['.ref-header','.ref-customer','.items','.ref-total','.ref-terms'];return {pages:document.querySelectorAll('.pagedjs_page').length,sectionsInside:!!r&&sections.every(selector=>{const b=page.querySelector(selector)?.getBoundingClientRect();return !!b&&b.left>=r.left&&b.right<=r.right&&b.top>=r.top&&b.bottom<=r.bottom})}})`, &preview, func(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) }),
	)
	if err != nil {
		t.Fatalf("print preview pagination failed: %v", err)
	}
	if preview.Pages != 1 || !preview.SectionsInside {
		t.Fatalf("print preview should fit on one A4 page without clipping: %+v", preview)
	}
}

func TestReferenceTemplateSingleItemWithShortNoteUsesAvailablePageHeight(t *testing.T) {
	notes := "ตรวจสอบรายการก่อนรับสินค้า"
	d := DocData{Type: "QUOTATION", DocumentNoFull: "REF-NOTE", DocumentDate: time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC), CustomerName: "Buyer", Notes: &notes,
		Items: []DocItem{{Description: "TEST_REG_20260902_PRODUCT", Quantity: 1, Unit: "ชิ้น", UnitPrice: 10, Amount: 10}}, Subtotal: 10, TotalAmount: 10}
	html, err := RenderReferenceDocumentCopies(d, StoreInfo{Name: "บริษัท ร้านค้าตัวอย่างและจัดจำหน่ายสินค้า จำกัด"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	pdf, err := htmlpdf.Render(context.Background(), html)
	if err != nil {
		t.Fatal(err)
	}
	if pages := bytes.Count(pdf, []byte("/Type /Page")) - bytes.Count(pdf, []byte("/Type /Pages")); pages != 1 {
		t.Fatalf("one-item PDF must fit on one A4 sheet, got %d", pages)
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
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	var result struct {
		Pages        int     `json:"pages"`
		HeightMM     float64 `json:"heightMM"`
		FooterInside bool    `json:"footerInside"`
	}
	err = chromedp.Run(ctx, chromedp.Navigate("about:blank"), chromedp.ActionFunc(func(ctx context.Context) error {
		tree, err := page.GetFrameTree().Do(ctx)
		if err != nil {
			return err
		}
		return page.SetDocumentContent(tree.Frame.ID, htmlpdf.PreparePreview(html)).Do(ctx)
	}), chromedp.Evaluate(`window.__posPagedReady.then(()=>{let p=document.querySelector('.pagedjs_page'),r=p.getBoundingClientRect(),f=p.querySelector('.ref-terms').getBoundingClientRect(),table=p.querySelector('.items').getBoundingClientRect();return {pages:document.querySelectorAll('.pagedjs_page').length,heightMM:table.height*25.4/96,footerInside:f.bottom<=r.bottom}})`, &result, func(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) }))
	if err != nil {
		t.Fatal(err)
	}
	if result.Pages != 1 || !result.FooterInside || result.HeightMM < 99.5 {
		t.Fatalf("one-item table should fill spare A4 space without adding pages: %+v", result)
	}
}

func TestReferenceTemplateSeveralItemsFitA4(t *testing.T) {
	for _, count := range []int{2, 4, 5} {
		t.Run(fmt.Sprintf("%d-items", count), func(t *testing.T) {
			items := make([]DocItem, count)
			for i := range items {
				items[i] = DocItem{Description: "สินค้าตัวอย่าง", Quantity: 1, Unit: "ชิ้น", UnitPrice: 100, Amount: 100}
			}
			d := DocData{Type: "QUOTATION", DocumentNoFull: "REF-MULTI", DocumentDate: time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC), CustomerName: "Buyer", Items: items, Subtotal: float64(count) * 100, TotalAmount: float64(count) * 100}
			html, err := RenderReferenceDocumentCopies(d, StoreInfo{Name: "Store"}, 0)
			if err != nil {
				t.Fatal(err)
			}
			pdf, err := htmlpdf.Render(context.Background(), html)
			if err != nil {
				t.Fatal(err)
			}
			pages := bytes.Count(pdf, []byte("/Type /Page")) - bytes.Count(pdf, []byte("/Type /Pages"))
			if pages != 1 {
				t.Fatalf("%d short items should remain on one page, got %d pages", count, pages)
			}
		})
	}
}

func TestReferenceTemplateLongDocumentPages(t *testing.T) {
	items := make([]DocItem, 60)
	for i := range items {
		items[i] = DocItem{Description: "Long item description with enough words to wrap across the table", Quantity: 1, Unit: "ชิ้น", UnitPrice: 10, Amount: 10}
	}
	d := DocData{Type: "INVOICE", DocumentNoFull: "REF-LONG", DocumentDate: time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC), CustomerName: "Buyer", Items: items, Subtotal: 600, TotalAmount: 600}
	html, err := RenderReferenceDocumentCopies(d, StoreInfo{Name: "Store"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	pdf, err := htmlpdf.Render(context.Background(), html)
	if err != nil {
		t.Fatal(err)
	}
	pages := bytes.Count(pdf, []byte("/Type /Page")) - bytes.Count(pdf, []byte("/Type /Pages"))
	if pages < 2 {
		t.Fatalf("long reference document should paginate, got %d pages", pages)
	}
}

func TestReferenceTemplateOnePagePDF(t *testing.T) {
	d := DocData{Type: "QUOTATION", DocumentNoFull: "REF-ONE", DocumentDate: time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC), CustomerName: "Buyer", Items: []DocItem{{Description: "Item", Quantity: 1, Unit: "ชิ้น", UnitPrice: 100, Amount: 100}}, Subtotal: 100, TotalAmount: 100}
	html, err := RenderReferenceDocumentCopies(d, StoreInfo{Name: "Store"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `data-compact="false"`) {
		t.Fatal("sparse one-item document should keep balanced item area")
	}
	pdf, err := htmlpdf.Render(context.Background(), html)
	if err != nil {
		t.Fatal(err)
	}
	if path := os.Getenv("REFERENCE_PDF_OUT"); path != "" {
		if err := os.WriteFile(path, pdf, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	pages := bytes.Count(pdf, []byte("/Type /Page")) - bytes.Count(pdf, []byte("/Type /Pages"))
	if pages != 1 {
		t.Fatalf("short reference document should fit on one page, got %d", pages)
	}
}
