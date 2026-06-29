package activity_log

import "testing"

func TestDeriveSeverity(t *testing.T) {
	cases := []struct {
		action, module, want string
	}{
		// Spec §10 examples.
		{"delete", "product", SeverityCritical},
		{"update", "product", SeverityHigh}, // price change
		{"create", "customer", SeverityMedium},
		{"print", "document", SeverityNormal},
		// Destructive on financial entities.
		{"void", "sale", SeverityCritical},
		{"cancel", "invoice", SeverityCritical},
		{"delete", "stock", SeverityCritical},
		// Security.
		{"manage-bank-accounts", "settings", SeverityCritical},
		{"manage-members", "settings", SeverityCritical},
		// Stock / financial movement.
		{"adjust", "stock", SeverityHigh},
		{"pay", "sale", SeverityHigh},
		{"receive", "purchasing", SeverityHigh},
		// Settings / purchasing edits.
		{"update", "settings", SeverityHigh},
		{"update", "purchasing", SeverityHigh},
		// Minor edits.
		{"update", "location", SeverityMedium},
		{"create", "product", SeverityNormal},
		// Unknown delete is still high.
		{"delete", "favourite", SeverityHigh},
	}
	for _, c := range cases {
		if got := DeriveSeverity(c.action, c.module); got != c.want {
			t.Errorf("DeriveSeverity(%q,%q)=%q want %q", c.action, c.module, got, c.want)
		}
	}
}

func TestDeriveCategory(t *testing.T) {
	cases := []struct {
		action, module, want string
	}{
		{"update", "product", CategoryInventory},
		{"adjust", "stock", CategoryInventory},
		{"create", "sale", CategorySales},
		{"convert", "invoice", CategorySales},
		{"update", "purchasing", CategoryPurchasing},
		{"update", "customer", CategoryCustomer},
		{"update", "promotion", CategoryPromotion},
		{"update", "settings", CategorySettings},
		{"manage-members", "settings", CategorySecurity},
		{"manage-bank-accounts", "settings", CategorySecurity},
		{"update", "expense", CategoryFinance},
		{"create", "system", CategoryGeneral},
	}
	for _, c := range cases {
		if got := DeriveCategory(c.action, c.module); got != c.want {
			t.Errorf("DeriveCategory(%q,%q)=%q want %q", c.action, c.module, got, c.want)
		}
	}
}

// Severity/category filters must agree with the badge a row shows: every tuple a
// filter selects must derive back to that same value.
func TestPairsMatchDerivation(t *testing.T) {
	for _, sev := range []string{SeverityCritical, SeverityHigh, SeverityMedium, SeverityNormal} {
		for _, p := range SeverityPairs(sev) {
			if got := DeriveSeverity(p[1], p[0]); got != sev {
				t.Errorf("SeverityPairs(%q) yielded (%q,%q) deriving %q", sev, p[0], p[1], got)
			}
		}
	}
	for _, cat := range []string{
		CategoryInventory, CategorySales, CategoryPurchasing, CategoryCustomer,
		CategoryPromotion, CategorySettings, CategorySecurity, CategoryFinance, CategoryGeneral,
	} {
		for _, p := range CategoryPairs(cat) {
			if got := DeriveCategory(p[1], p[0]); got != cat {
				t.Errorf("CategoryPairs(%q) yielded (%q,%q) deriving %q", cat, p[0], p[1], got)
			}
		}
	}
}
