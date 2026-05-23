package invoice

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"

	"pos-backend/internal/modules/auth"
)

//go:embed assets/DejaVuSansCondensed.ttf
var embeddedDejaVuSans []byte

//go:embed assets/Tahoma.ttf
var embeddedTahoma []byte

type CustomerBenefitResolver interface {
	Resolve(ctx context.Context, storeID, customerID string) (level int, discountPercent float64, err error)
}

type Service struct {
	repo     Repository
	resolver CustomerBenefitResolver
	storage  PaymentProofStorage
}

func NewService(repo Repository, resolver CustomerBenefitResolver, storage PaymentProofStorage) Service {
	if storage == nil {
		storage = NoopPaymentProofStorage{}
	}
	return Service{repo: repo, resolver: resolver, storage: storage}
}

func (s Service) Create(ctx context.Context, actor auth.Claims, storeID string, req CreateInvoiceRequest) (Invoice, error) {
	if err := validateCreateRequest(req); err != nil {
		return Invoice{}, err
	}
	allowed, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Invoice{}, err
	}
	if !allowed {
		return Invoice{}, ErrForbiddenStoreAccess
	}
	if s.resolver == nil {
		return Invoice{}, ErrCustomerNotFound
	}

	customerID := strings.TrimSpace(req.CustomerID)
	level, networkDiscountPercent, err := s.resolver.Resolve(ctx, storeID, customerID)
	if err != nil {
		return Invoice{}, ErrCustomerNotFound
	}

	now := time.Now().UTC()
	invoice := Invoice{
		ID:                     newID(),
		StoreID:                storeID,
		InvoiceNumber:          newInvoiceNumber(now),
		CustomerID:             customerID,
		CashierUserID:          actor.UserID,
		Status:                 StatusUnpaid,
		Note:                   strings.TrimSpace(req.Note),
		DueAt:                  req.DueAt,
		VATIncluded:            true,
		VATPercent:             7,
		CustomerLevel:          &level,
		NetworkDiscountPercent: networkDiscountPercent,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	if req.VATIncluded != nil {
		invoice.VATIncluded = *req.VATIncluded
	}
	if req.VATPercent != nil {
		invoice.VATPercent = *req.VATPercent
	}

	for _, item := range req.Items {
		invoice.Items = append(invoice.Items, InvoiceItem{
			ProductID:     strings.TrimSpace(item.ProductID),
			Quantity:      item.Quantity,
			DiscountType:  item.DiscountType,
			DiscountValue: item.DiscountValue,
		})
		invoice.TotalItems += item.Quantity
	}
	return s.repo.Create(ctx, invoice)
}

func (s Service) ListByStore(ctx context.Context, actor auth.Claims, storeID string) ([]Invoice, error) {
	allowed, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbiddenStoreAccess
	}
	return s.repo.ListByStore(ctx, storeID)
}

func (s Service) GetByID(ctx context.Context, actor auth.Claims, storeID, invoiceID string) (Invoice, error) {
	allowed, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Invoice{}, err
	}
	if !allowed {
		return Invoice{}, ErrForbiddenStoreAccess
	}
	return s.repo.GetByID(ctx, storeID, invoiceID)
}

func (s Service) AddPayment(ctx context.Context, actor auth.Claims, storeID, invoiceID string, req CreateInvoicePaymentRequest) (Invoice, error) {
	if strings.TrimSpace(req.PaymentMethod) == "" {
		return Invoice{}, ErrInvalidPaymentMethod
	}
	if req.PaidAmount <= 0 {
		return Invoice{}, ErrInvalidPaidAmount
	}

	allowed, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Invoice{}, err
	}
	if !allowed {
		return Invoice{}, ErrForbiddenStoreAccess
	}

	now := time.Now().UTC()
	proofURL, proofMimeType, proofFileName, err := s.storage.SavePaymentProof(req.ProofFile)
	if err != nil {
		return Invoice{}, err
	}
	payment := InvoicePayment{
		ID:            newInvoicePaymentID(),
		InvoiceID:     invoiceID,
		PaidAmount:    req.PaidAmount,
		PaymentMethod: strings.TrimSpace(req.PaymentMethod),
		Note:          strings.TrimSpace(req.Note),
		ProofURL:      proofURL,
		ProofMimeType: proofMimeType,
		ProofFileName: proofFileName,
		PaidAt:        now,
		CreatedAt:     now,
	}
	return s.repo.AddPayment(ctx, storeID, invoiceID, payment)
}

func (s Service) MarkUnpaid(ctx context.Context, actor auth.Claims, storeID, invoiceID string, req MarkInvoiceUnpaidRequest) (Invoice, error) {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return Invoice{}, ErrInvalidUnpayReason
	}

	allowed, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return Invoice{}, err
	}
	if !allowed {
		return Invoice{}, ErrForbiddenStoreAccess
	}

	return s.repo.MarkUnpaid(ctx, storeID, invoiceID, actor.UserID, reason, time.Now().UTC())
}

func (s Service) GetPaymentProof(ctx context.Context, actor auth.Claims, storeID, invoiceID, paymentID string) (InvoicePayment, error) {
	allowed, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return InvoicePayment{}, err
	}
	if !allowed {
		return InvoicePayment{}, ErrForbiddenStoreAccess
	}

	payment, err := s.repo.GetPaymentProof(ctx, storeID, invoiceID, paymentID)
	if err != nil {
		return InvoicePayment{}, err
	}
	if strings.TrimSpace(payment.ProofURL) == "" {
		return InvoicePayment{}, ErrPaymentProofNotFound
	}
	return payment, nil
}

func (s Service) GeneratePDF(ctx context.Context, actor auth.Claims, storeID, invoiceID string) ([]byte, error) {
	inv, err := s.GetByID(ctx, actor, storeID, invoiceID)
	if err != nil {
		return nil, err
	}
	return renderInvoicePDF(inv)
}

func validateCreateRequest(req CreateInvoiceRequest) error {
	if strings.TrimSpace(req.CustomerID) == "" {
		return ErrCustomerNotFound
	}
	if len(req.Items) == 0 {
		return ErrInvalidInvoiceItems
	}
	if req.VATPercent != nil && (*req.VATPercent < 0 || *req.VATPercent > 100) {
		return ErrInvalidVATPercent
	}
	for _, item := range req.Items {
		if strings.TrimSpace(item.ProductID) == "" || item.Quantity <= 0 {
			return ErrInvalidInvoiceItem
		}
		discountType := normalizeDiscountType(item.DiscountType)
		if discountType != "" && discountType != discountTypeAmount && discountType != discountTypePercent {
			return ErrInvalidDiscountType
		}
		if discountType == "" && item.DiscountValue != nil {
			return ErrInvalidDiscountType
		}
		if discountType != "" && item.DiscountValue == nil {
			return ErrDiscountValueRequired
		}
		if item.DiscountValue != nil && *item.DiscountValue < 0 {
			return ErrInvalidDiscountValue
		}
		if discountType == discountTypePercent && item.DiscountValue != nil && *item.DiscountValue > 100 {
			return ErrInvalidPercentDiscount
		}
	}
	return nil
}

func renderInvoicePDF(inv Invoice) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	fontFamily := registerPDFUnicodeFont(pdf)
	pdf.SetMargins(10, 10, 10)
	pdf.SetAutoPageBreak(true, 10)

	pdf.SetFont(fontFamily, "", 17)
	pdf.CellFormat(190, 10, "Outstanding Invoice", "", 1, "L", false, 0, "")

	pdf.SetFont(fontFamily, "", 11)
	pdf.CellFormat(95, 7, fmt.Sprintf("Invoice No: %s", inv.InvoiceNumber), "", 0, "L", false, 0, "")
	pdf.CellFormat(95, 7, fmt.Sprintf("Status: %s", strings.ToUpper(inv.Status)), "", 1, "R", false, 0, "")
	pdf.CellFormat(95, 7, fmt.Sprintf("Store: %s", safeText(inv.StoreName)), "", 0, "L", false, 0, "")
	pdf.CellFormat(95, 7, fmt.Sprintf("Customer: %s", safeText(inv.CustomerName)), "", 1, "R", false, 0, "")
	pdf.CellFormat(95, 7, fmt.Sprintf("Created: %s", inv.CreatedAt.Format("2006-01-02 15:04")), "", 0, "L", false, 0, "")
	if inv.DueAt != nil {
		pdf.CellFormat(95, 7, fmt.Sprintf("Due: %s", inv.DueAt.Format("2006-01-02")), "", 1, "R", false, 0, "")
	} else {
		pdf.CellFormat(95, 7, "Due: -", "", 1, "R", false, 0, "")
	}
	pdf.Ln(10)

	colQty := 16.0
	colItem := 84.0
	colUnit := 30.0
	colDiscount := 30.0
	colTotal := 30.0

	pdf.SetFont(fontFamily, "", 10)
	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(colQty, 8, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colItem, 8, "Item", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colUnit, 8, "Unit", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colDiscount, 8, "Discount", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colTotal, 8, "Line Total", "1", 1, "C", true, 0, "")

	for _, it := range inv.Items {
		pdf.CellFormat(colQty, 7, fmt.Sprintf("%d", it.Quantity), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colItem, 7, truncateText(it.ProductName, 38), "1", 0, "L", false, 0, "")
		pdf.CellFormat(colUnit, 7, fmt.Sprintf("%.2f", it.UnitPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colDiscount, 7, fmt.Sprintf("%.2f", it.LineDiscountTotal), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colTotal, 7, fmt.Sprintf("%.2f", it.LineTotal), "1", 1, "R", false, 0, "")
	}

	pdf.Ln(6)
	pdf.SetFont(fontFamily, "", 11)
	x := 110.0
	pdf.SetX(x)
	pdf.CellFormat(40, 7, "Subtotal", "1", 0, "L", false, 0, "")
	pdf.CellFormat(50, 7, fmt.Sprintf("%.2f", inv.SubtotalAmount), "1", 1, "R", false, 0, "")
	pdf.SetX(x)
	pdf.CellFormat(40, 7, "Discount", "1", 0, "L", false, 0, "")
	pdf.CellFormat(50, 7, fmt.Sprintf("%.2f", inv.DiscountAmount), "1", 1, "R", false, 0, "")
	pdf.SetX(x)
	pdf.CellFormat(40, 7, fmt.Sprintf("VAT %.2f%%", inv.VATPercent), "1", 0, "L", false, 0, "")
	pdf.CellFormat(50, 7, fmt.Sprintf("%.2f", inv.VATAmount), "1", 1, "R", false, 0, "")
	pdf.SetX(x)
	vatMode := "Included"
	if !inv.VATIncluded {
		vatMode = "Excluded"
	}
	pdf.CellFormat(40, 7, "VAT Mode", "1", 0, "L", false, 0, "")
	pdf.CellFormat(50, 7, vatMode, "1", 1, "R", false, 0, "")
	pdf.SetX(x)
	pdf.CellFormat(40, 7, "Total", "1", 0, "L", false, 0, "")
	pdf.CellFormat(50, 7, fmt.Sprintf("%.2f", inv.TotalAmount), "1", 1, "R", false, 0, "")
	pdf.SetX(x)
	pdf.CellFormat(40, 7, "Paid", "1", 0, "L", false, 0, "")
	pdf.CellFormat(50, 7, fmt.Sprintf("%.2f", inv.PaidAmount), "1", 1, "R", false, 0, "")
	pdf.SetX(x)
	pdf.CellFormat(40, 7, "Remaining", "1", 0, "L", false, 0, "")
	pdf.CellFormat(50, 7, fmt.Sprintf("%.2f", inv.RemainingAmount), "1", 1, "R", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func registerPDFUnicodeFont(pdf *gofpdf.Fpdf) string {
	if len(embeddedTahoma) > 0 {
		pdf.AddUTF8FontFromBytes("UnicodeTH", "", embeddedTahoma)
		if pdf.Ok() {
			return "UnicodeTH"
		}
		pdf.ClearError()
	}

	if len(embeddedDejaVuSans) > 0 {
		pdf.AddUTF8FontFromBytes("Unicode", "", embeddedDejaVuSans)
		if pdf.Ok() {
			return "Unicode"
		}
		pdf.ClearError()
	}

	if path := strings.TrimSpace(os.Getenv("PDF_TTF_PATH")); path != "" {
		if _, err := os.Stat(path); err == nil {
			pdf.AddUTF8Font("Unicode", "", path)
			if pdf.Ok() {
				return "Unicode"
			}
			pdf.ClearError()
		}
	}

	fallbacks := []string{
		"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
		"/System/Library/Fonts/Supplemental/Tahoma.ttf",
		"/Library/Fonts/Arial Unicode.ttf",
		"/Library/Fonts/Tahoma.ttf",
		"/System/Library/Fonts/Supplemental/Thonburi.ttf",
		"/Library/Fonts/Thonburi.ttf",
		"/usr/share/fonts/truetype/noto/NotoSansThai-Regular.ttf",
		"/usr/share/fonts/truetype/noto/NotoSansThaiUI-Regular.ttf",
	}
	for _, path := range fallbacks {
		if _, err := os.Stat(path); err == nil {
			pdf.AddUTF8Font("Unicode", "", path)
			if pdf.Ok() {
				return "Unicode"
			}
			pdf.ClearError()
		}
	}

	return "Arial"
}

func safeText(v string) string {
	if strings.TrimSpace(v) == "" {
		return "-"
	}
	return v
}

func truncateText(v string, max int) string {
	runes := []rune(v)
	if len(runes) <= max {
		return v
	}
	return string(runes[:max-1]) + "…"
}
