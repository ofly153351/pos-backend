package creditsale

import (
	"context"
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
func (s Service) ensureAccess(ctx context.Context, actor auth.Claims, storeID string) error {
	if strings.TrimSpace(storeID) == "" {
		return ErrStoreIDRequired
	}
	ok, err := s.repo.UserCanOperateStore(ctx, storeID, actor.UserID, actor.Role)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s Service) List(ctx context.Context, actor auth.Claims, storeID string) ([]CreditSale, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, storeID)
}

func (s Service) Get(ctx context.Context, actor auth.Claims, storeID, creditSaleID string) (CreditSale, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return CreditSale{}, err
	}
	return s.repo.Get(ctx, storeID, creditSaleID)
}

func (s Service) Summary(ctx context.Context, actor auth.Claims, storeID string) (DebtSummary, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return DebtSummary{}, err
	}
	return s.repo.Summary(ctx, storeID)
}

// Create mints a REAL product-backed sale (deducts stock, accrual revenue) tagged
// payment_method='credit', then records the receivable + an optional down-payment.
func (s Service) Create(ctx context.Context, actor auth.Claims, storeID string, req CreateCreditSaleRequest) (CreditSale, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return CreditSale{}, err
	}
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

	// Build and create the underlying sale (no VAT — matches the store's POS config).
	vatIncluded := false
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
		PaymentMethod: creditPaymentMethod,
		PaidAmount:    roundMoney(req.DownPayment),
		VATIncluded:   &vatIncluded,
		Note:          strings.TrimSpace(req.Note),
		CustomerID:    strings.TrimSpace(req.CustomerID),
		Items:         saleItems,
	})
	if err != nil {
		return CreditSale{}, err
	}

	// The sale is committed (stock deducted); compute the receivable from its total.
	downPayment := roundMoney(req.DownPayment)
	if downPayment > createdSale.TotalAmount {
		downPayment = createdSale.TotalAmount // clamp; never fail post-commit (avoids orphan)
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
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return CreditSale{}, err
	}
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

// Cancel restocks the goods and voids the underlying sale, removing its revenue and
// COGS from finance reports. No bad-debt expense is booked — the goods came back.
func (s Service) Cancel(ctx context.Context, actor auth.Claims, storeID, creditSaleID string) (CreditSale, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return CreditSale{}, err
	}
	return s.repo.Cancel(ctx, storeID, creditSaleID, actor.UserID)
}

// Statement renders a customer statement PDF for the credit sale's customer,
// reusing the existing docpdf.RenderStatementPDF renderer. Rows are built from
// credit_sales (one per receivable) and the payment timeline is summarised in the
// note. No new PDF engine is introduced.
func (s Service) Statement(ctx context.Context, actor auth.Claims, storeID, creditSaleID string) ([]byte, error) {
	if err := s.ensureAccess(ctx, actor, storeID); err != nil {
		return nil, err
	}
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
