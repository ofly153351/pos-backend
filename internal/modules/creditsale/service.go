package creditsale

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
	"pos-backend/internal/modules/sale"
	"pos-backend/internal/platform/docpdf"
)

type Service struct {
	repo        Repository
	saleService sale.Service
}

func NewService(repo Repository, saleService sale.Service) Service {
	return Service{repo: repo, saleService: saleService}
}

// Access: any store member (owner/manager/cashier) may grant credit and record
// payments — mirrors the expense create policy.

func (s Service) List(ctx context.Context, actor auth.Claims, storeID string) ([]CreditSale, error) {
	return s.repo.List(ctx, storeID)
}

func (s Service) Get(ctx context.Context, actor auth.Claims, storeID, creditSaleID string) (CreditSale, error) {
	return s.repo.Get(ctx, storeID, creditSaleID)
}

func (s Service) Summary(ctx context.Context, actor auth.Claims, storeID string) (DebtSummary, error) {
	return s.repo.Summary(ctx, storeID)
}

func (s Service) Aging(ctx context.Context, actor auth.Claims, storeID string) (AgingSummary, error) {
	return s.repo.Aging(ctx, storeID)
}

// Create mints a REAL product-backed sale (deducts stock, accrual revenue) tagged
// payment_method='credit', then records the receivable + an optional down-payment.
func (s Service) Create(ctx context.Context, actor auth.Claims, storeID string, req CreateCreditSaleRequest) (CreditSale, error) {
	if strings.TrimSpace(req.CustomerID) == "" {
		return CreditSale{}, ErrCustomerRequired
	}
	if len(req.Items) == 0 {
		return CreditSale{}, ErrNoItems
	}
	for _, it := range req.Items {
		if strings.TrimSpace(it.ProductID) == "" || it.Quantity <= 0 {
			return CreditSale{}, ErrInvalidItem
		}
	}
	if req.DownPayment < 0 {
		return CreditSale{}, ErrInvalidDownPayment
	}
	saleType := strings.TrimSpace(req.Type)
	switch saleType {
	case "":
		saleType = typeCredit
	case typeCredit, typeLoan:
	default:
		return CreditSale{}, ErrInvalidType
	}
	// A loan (ยืมสินค้า) is settled by returning goods, not cash, and ReturnGoods force-
	// settles the full total on a complete return — a cash down payment would be silently
	// absorbed with no refund path. Reject it.
	if saleType == typeLoan && req.DownPayment > 0 {
		return CreditSale{}, ErrInvalidDownPayment
	}

	// Build the underlying sale. Discount/VAT/location intent flows through verbatim
	// so the receivable total matches exactly what the cashier saw (POS "open credit
	// bill" and the credit-sales form both feed these). When VATPercent/VATIncluded
	// are nil the sale falls back to the store's POS defaults.
	saleItems := make([]sale.CreateSaleItemRequest, 0, len(req.Items))
	for _, it := range req.Items {
		saleItems = append(saleItems, sale.CreateSaleItemRequest{
			ProductID:     strings.TrimSpace(it.ProductID),
			Quantity:      it.Quantity,
			DiscountType:  it.DiscountType,
			DiscountValue: it.DiscountValue,
		})
	}
	createdSale, err := s.saleService.Create(ctx, actor, storeID, sale.CreateSaleRequest{
		PaymentMethod:  creditPaymentMethod,
		LocationID:     strings.TrimSpace(req.LocationID),
		IdempotencyKey: strings.TrimSpace(req.IdempotencyKey),
		PaidAmount:     roundMoney(req.DownPayment),
		ManualDiscount: req.BillDiscount,
		PromoDiscount:  req.PromoDiscount,
		PromotionIDs:   req.PromotionIDs,
		VATIncluded:    req.VATIncluded,
		VATPercent:     req.VATPercent,
		Note:           strings.TrimSpace(req.Note),
		CustomerID:     strings.TrimSpace(req.CustomerID),
		Items:          saleItems,
	})
	if err != nil {
		if errors.Is(err, sale.ErrInvalidPaidAmount) && roundMoney(req.DownPayment) > 0 {
			// The sale repository validates the final VAT/discounted total before
			// deduction. Translate its generic paid-amount error into the credit
			// API's field-specific 422 instead of leaking a 500.
			return CreditSale{}, ErrInvalidDownPayment
		}
		return CreditSale{}, err
	}

	// Idempotent replay: when the underlying sale was deduped by its Idempotency-Key,
	// a receivable already exists for it — return that instead of minting a duplicate
	// credit_sales row for the same sale.
	if existing, found, ferr := s.repo.FindBySaleID(ctx, storeID, createdSale.ID); ferr == nil && found {
		return existing, nil
	}

	// The sale service now rejects an excessive credit down payment before the
	// transaction reaches this post-commit clamp. Keep this guard as a defensive
	// invariant for direct repository callers and old integrations.
	downPayment := roundMoney(req.DownPayment)
	if downPayment > createdSale.TotalAmount {
		return CreditSale{}, ErrInvalidDownPayment
	}
	remaining := roundMoney(createdSale.TotalAmount - downPayment)
	if remaining < 0 {
		remaining = 0
	}
	status := StatusPending
	if remaining <= 0 {
		status = StatusCompleted
	} else if downPayment > 0 {
		status = StatusPartial
	}

	now := time.Now().UTC().Format(time.RFC3339)
	cs := CreditSale{
		ID:              newCreditSaleID(),
		StoreID:         storeID,
		SaleID:          createdSale.ID,
		Type:            saleType,
		CustomerID:      strings.TrimSpace(req.CustomerID),
		TotalAmount:     createdSale.TotalAmount,
		PaidAmount:      downPayment,
		RemainingAmount: remaining,
		Status:          status,
		DueDate:         strings.TrimSpace(req.DueDate),
		Note:            strings.TrimSpace(req.Note),
		CreatedBy:       actor.UserID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	var initial *CreditPayment
	if downPayment > 0 {
		initial = &CreditPayment{
			ID:           newCreditPaymentID(),
			CreditSaleID: cs.ID,
			StoreID:      storeID,
			Amount:       downPayment,
			Method:       "cash",
			Note:         "down payment",
			PaidAt:       now,
			CreatedBy:    actor.UserID,
		}
	}
	return s.repo.CreateReceivable(ctx, cs, initial)
}

func (s Service) AddPayment(ctx context.Context, actor auth.Claims, storeID, creditSaleID string, req AddPaymentRequest) (CreditSale, error) {
	amount := roundMoney(req.Amount)
	if amount <= 0 {
		return CreditSale{}, ErrInvalidAmount
	}
	method := strings.TrimSpace(req.Method)
	if method == "" {
		method = "cash"
	}
	p := CreditPayment{
		ID:           newCreditPaymentID(),
		CreditSaleID: creditSaleID,
		StoreID:      storeID,
		Amount:       amount,
		Method:       method,
		Note:         strings.TrimSpace(req.Note),
		PaidAt:       time.Now().UTC().Format(time.RFC3339),
		CreatedBy:    actor.UserID,
	}
	return s.repo.AddPayment(ctx, storeID, p)
}

// ReturnGoods restocks borrowed goods (loan type only) and settles the receivable by the
// value of what came back. Any store member may record a return (mirrors AddPayment).
func (s Service) ReturnGoods(ctx context.Context, actor auth.Claims, storeID, creditSaleID string, req ReturnGoodsRequest) (CreditSale, error) {
	if len(req.Items) == 0 {
		return CreditSale{}, ErrNoReturnItems
	}
	for _, it := range req.Items {
		if strings.TrimSpace(it.ProductID) == "" || it.Quantity <= 0 {
			return CreditSale{}, ErrNoReturnItems
		}
	}
	return s.repo.ReturnGoods(ctx, storeID, creditSaleID, actor.UserID, req.Items)
}

// Cancel restocks the goods and voids the underlying sale, removing its revenue and
// COGS from finance reports. No bad-debt expense is booked — the goods came back.
func (s Service) Cancel(ctx context.Context, actor auth.Claims, storeID, creditSaleID string) (CreditSale, error) {
	return s.repo.Cancel(ctx, storeID, creditSaleID, actor.UserID)
}

// Statement renders a customer statement PDF for the credit sale's customer,
// reusing the existing docpdf.RenderStatementPDF renderer. Rows are built from
// credit_sales (one per receivable) and the payment timeline is summarised in the
// note. No new PDF engine is introduced.
func (s Service) Statement(ctx context.Context, actor auth.Claims, storeID, creditSaleID string) ([]byte, error) {
	sc, err := s.repo.StatementContext(ctx, storeID, creditSaleID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	earliest := now
	rows := make([]docpdf.StatementInvoiceRow, 0, len(sc.Sales))
	noteLines := make([]string, 0)
	for _, cs := range sc.Sales {
		issue := parseTimestamp(cs.CreatedAt)
		if !issue.IsZero() && issue.Before(earliest) {
			earliest = issue
		}
		rows = append(rows, docpdf.StatementInvoiceRow{
			InvoiceNo: cs.DocumentNumber,
			IssueDate: issue,
			DueDate:   parseDateOnly(cs.DueDate),
			Amount:    cs.TotalAmount,
			Paid:      cs.PaidAmount,
			Balance:   cs.RemainingAmount,
			Status:    statementStatus(cs.Status),
		})
		for _, p := range cs.Payments {
			noteLines = append(noteLines, fmt.Sprintf("%s — %s — %.2f (%s)", cs.DocumentNumber, shortDate(p.PaidAt), p.Amount, p.Method))
		}
	}

	note := ""
	if len(noteLines) > 0 {
		note = "ประวัติการชำระเงิน / Payment history:\n" + strings.Join(noteLines, "\n")
	}

	return docpdf.RenderStatementPDF(docpdf.StatementPDFInput{
		SellerName:      sc.StoreName,
		SellerAddress:   sc.StoreAddress,
		SellerTaxID:     sc.StoreTaxID,
		CustomerName:    sc.CustomerName,
		CustomerAddress: sc.CustomerAddr,
		StatementNo:     sc.DocumentNo,
		IssueDate:       now,
		PeriodStart:     earliest,
		PeriodEnd:       now,
		Invoices:        rows,
		Note:            note,
	})
}

func statementStatus(status string) string {
	switch status {
	case StatusCompleted:
		return "paid"
	case StatusOverdue:
		return "overdue"
	default:
		return "outstanding"
	}
}

func parseTimestamp(v string) time.Time {
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t
	}
	return time.Time{}
}

func parseDateOnly(v string) time.Time {
	if t, err := time.Parse("2006-01-02", v); err == nil {
		return t
	}
	return time.Time{}
}

func shortDate(v string) string {
	if t := parseTimestamp(v); !t.IsZero() {
		return t.Format("02/01/2006")
	}
	return v
}
