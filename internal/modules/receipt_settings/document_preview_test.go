package receipt_settings

import (
	"context"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"pos-backend/internal/modules/auth"
	"pos-backend/internal/modules/store"
	"pos-backend/internal/platform/dochtml"
)

type previewStoreRepository struct {
	store.Repository
	store     store.Store
	err       error
	requested []string
}

func (r *previewStoreRepository) GetByID(_ context.Context, id string) (store.Store, error) {
	r.requested = append(r.requested, id)
	return r.store, r.err
}

type previewSettingsRepository struct {
	settings ReceiptSettings
	err      error
	writes   int
}

func (r *previewSettingsRepository) GetByStoreID(context.Context, string) (ReceiptSettings, error) {
	return r.settings, r.err
}
func (r *previewSettingsRepository) Insert(_ context.Context, s ReceiptSettings) (ReceiptSettings, error) {
	r.writes++
	return s, nil
}
func (r *previewSettingsRepository) Upsert(_ context.Context, s ReceiptSettings) (ReceiptSettings, error) {
	r.writes++
	return s, nil
}

func TestDocumentTemplatePreviewWithoutDocumentsOrStoreData(t *testing.T) {
	for _, templateKey := range []string{"standard", "classic", "template3"} {
		t.Run(templateKey, func(t *testing.T) {
			repo := &previewSettingsRepository{err: ErrNotFound}
			stores := &previewStoreRepository{store: store.Store{ID: "empty-store", Name: "Current Store Identity", Address: "987 Current Address", Phone: "02-123-5678", Email: "owner@example.test", Fax: "02-987-6543", Website: "store.example.test", TaxID: "1234567890123", PromptPayID: "REAL-PAYMENT-IDENTIFIER", LogoURL: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/lS8AAAAASUVORK5CYII="}}
			service := Service{repo: repo, storeRepo: stores} // no customers, products or documents
			html, err := service.PreviewDocumentHTML(context.Background(), auth.Claims{}, "empty-store", UpdateReceiptSettingsRequest{DocumentTemplateKey: &templateKey})
			if err != nil {
				t.Fatal(err)
			}
			if repo.writes != 0 {
				t.Fatalf("preview wrote %d settings rows", repo.writes)
			}
			if len(stores.requested) != 1 || stores.requested[0] != "empty-store" {
				t.Fatalf("expected current store lookup, got %v", stores.requested)
			}
			for _, want := range []string{"QUO-PREVIEW", "Current Store Identity", "987 Current Address", "02-123-5678", "owner@example.test", "02-987-6543", "store.example.test", "1234567890123", "ตัวอย่างลูกค้า", "สินค้าตัวอย่าง", "data:image/png;base64,", `data-show-signature="true"`, "ต้นฉบับ"} {
				if !strings.Contains(string(html), want) {
					t.Errorf("preview HTML missing %q", want)
				}
			}
			if strings.Contains(string(html), "บริษัทตัวอย่าง") || strings.Contains(string(html), "REAL-PAYMENT-IDENTIFIER") {
				t.Error("preview included sample store or real payment information")
			}
		})
	}
}

func TestDocumentTemplatePreviewMissingStoreIdentity(t *testing.T) {
	repo := &previewSettingsRepository{err: ErrNotFound}
	stores := &previewStoreRepository{store: store.Store{ID: "current", Name: "Only Name"}}
	service := Service{repo: repo, storeRepo: stores}
	html, err := service.PreviewDocumentHTML(context.Background(), auth.Claims{}, "current", UpdateReceiptSettingsRequest{})
	if err != nil || !strings.Contains(string(html), "Only Name") {
		t.Fatalf("partial identity should render: %v", err)
	}
	if strings.Contains(string(html), "บริษัทตัวอย่าง") || strings.Contains(string(html), "456 ถนนสาธิต") {
		t.Fatal("partial identity must not be filled from fixture")
	}
	stores.err = errors.New("store read failed")
	if _, err := service.PreviewDocumentHTML(context.Background(), auth.Claims{}, "current", UpdateReceiptSettingsRequest{}); !errors.Is(err, stores.err) {
		t.Fatalf("store read error must surface, got %v", err)
	}
}

func TestDocumentTemplatePreviewRefreshesStoreIdentity(t *testing.T) {
	repo := &previewSettingsRepository{err: ErrNotFound}
	stores := &previewStoreRepository{store: store.Store{ID: "current", Name: "First Store"}}
	service := Service{repo: repo, storeRepo: stores}
	first, err := service.PreviewDocumentHTML(context.Background(), auth.Claims{}, "current", UpdateReceiptSettingsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	stores.store = store.Store{ID: "current", Name: "Updated Store"}
	second, err := service.PreviewDocumentHTML(context.Background(), auth.Claims{}, "current", UpdateReceiptSettingsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(first), "First Store") || strings.Contains(string(second), "First Store") || !strings.Contains(string(second), "Updated Store") {
		t.Fatal("store identity was not refreshed on each preview")
	}
}

func TestRealDocumentRenderDoesNotGainPreviewOnlyStoreContact(t *testing.T) {
	doc := documentPreviewFixture()
	actualStore := dochtml.StoreInfo{Name: "Real Store", Email: "real@example.test"}
	for _, render := range []func(dochtml.DocData, dochtml.StoreInfo, int) (string, error){
		dochtml.RenderUnifiedDocumentCopies, dochtml.RenderReferenceDocumentCopies, dochtml.RenderTemplate3DocumentCopies,
	} {
		html, err := render(doc, actualStore, 0)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(html, "real@example.test") {
			t.Fatal("preview-only contact appeared in a real document")
		}
	}
}

func TestDocumentPreviewEmbedsOnlyCurrentStoreLogo(t *testing.T) {
	png := []byte{137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 73, 72, 68, 82, 0, 0, 0, 1, 0, 0, 0, 1}
	image := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(png)
	}))
	defer image.Close()
	stores := &previewStoreRepository{store: store.Store{ID: "current", Name: "Current", LogoURL: image.URL}}
	service := Service{repo: &previewSettingsRepository{err: ErrNotFound}, storeRepo: stores}
	html, err := service.PreviewDocumentHTML(context.Background(), auth.Claims{}, "current", UpdateReceiptSettingsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "data:image/png;base64,") || strings.Contains(string(html), image.URL) {
		t.Fatal("store logo was not embedded in preview")
	}
	if got := previewStoreLogo(context.Background(), "javascript:alert(1)"); got != "" {
		t.Fatalf("unsafe logo URL: %q", got)
	}
}

func TestDocumentPreviewFixtureTotalsMatchEveryMockItem(t *testing.T) {
	doc := documentPreviewFixture()
	if len(doc.Items) != 38 {
		t.Fatalf("mock item count: %d", len(doc.Items))
	}
	subtotal, discount, amount := 0.0, 0.0, 0.0
	for _, item := range doc.Items {
		subtotal += item.Quantity * item.UnitPrice
		discount += item.DiscountValue
		amount += item.Amount
	}
	if doc.Subtotal != subtotal || doc.TotalDiscount != discount || doc.PreVatAmount != amount || doc.VatAmount != math.Round(amount*doc.VatRate)/100 || doc.TotalAmount != amount+doc.VatAmount || doc.PaymentAmount != doc.TotalAmount {
		t.Fatalf("inconsistent mock amounts: subtotal=%v discount=%v amount=%v VAT=%v total=%v", doc.Subtotal, doc.TotalDiscount, doc.PreVatAmount, doc.VatAmount, doc.TotalAmount)
	}
}

func TestDocumentTemplatePreviewResponseRunsExistingPagedJS(t *testing.T) {
	repo := &previewSettingsRepository{err: ErrNotFound}
	stores := &previewStoreRepository{store: store.Store{ID: "current", Name: "Actual Store"}}
	app := fiber.New()
	app.Post("/api/stores/:storeID/receipt-settings/document-preview", NewHandler(Service{repo: repo, storeRepo: stores}).DocumentPreviewHTML)
	req := httptest.NewRequest("POST", "/api/stores/current/receipt-settings/document-preview", strings.NewReader(`{"document_template_key":"standard"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("preview status: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"Actual Store", "__posPagedReady", "PagedPolyfill.preview()", "Sample product 38"} {
		if !strings.Contains(string(body), expected) {
			t.Errorf("missing %q from preview response", expected)
		}
	}
	if repo.writes != 0 {
		t.Fatalf("preview wrote %d settings rows", repo.writes)
	}
}
