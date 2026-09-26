package document

import (
	"strings"
	"testing"
	"time"
)

func TestToDocData_BillRowsComeFromSelectedDeliveryOrders(t *testing.T) {
	due := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	doc := &Document{
		Type:         TypeBill,
		DocumentDate: time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC),
		DueDate:      &due,
		Items: []DocumentItem{
			{Description: "DO256909-0001", Amount: 512.53},
			{Description: "DO256909-0002", Amount: 1200},
		},
	}

	got := toDocData(doc)
	if len(got.BillRows) != 2 {
		t.Fatalf("expected 2 bill rows, got %d", len(got.BillRows))
	}
	if got.BillRows[0].DocumentNo != "DO256909-0001" || got.BillRows[0].Amount != 512.53 {
		t.Fatalf("unexpected first bill row: %+v", got.BillRows[0])
	}
	if got.BillRows[0].DueDate == nil || !strings.HasPrefix(got.BillRows[0].DueDate.Format("2006-01-02"), "2026-09-30") {
		t.Fatalf("bill row due date was not carried over")
	}
}
