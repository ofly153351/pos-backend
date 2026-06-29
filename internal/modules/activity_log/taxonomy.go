package activity_log

// taxonomy.go is the single source of truth for the *derived* intelligence of an
// activity: its business severity and its category. Both are pure functions of the
// already-stored (action, module) pair — no schema column, no LLM, no drift. The
// read-time DTO mapper derives them for display, and the repository derives the
// filter predicates from the SAME functions (see SeverityPairs / CategoryPairs), so
// a severity/category filter can never disagree with the badge a row shows.

// Severity levels, most → least important. AI never decides these; they are a
// deterministic mapping a store owner can reason about ("deleting a sale is
// critical, editing a category is medium").
const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityNormal   = "normal"
)

// Category buckets mirror the spec's icon set (Inventory/Finance/Sales/Purchasing/
// Customer/Promotion/Settings/Security). "general" is the catch-all for anything
// the middleware could not classify.
const (
	CategoryInventory  = "inventory"
	CategorySales      = "sales"
	CategoryPurchasing = "purchasing"
	CategoryCustomer   = "customer"
	CategoryPromotion  = "promotion"
	CategorySettings   = "settings"
	CategorySecurity   = "security"
	CategoryFinance    = "finance"
	CategoryGeneral    = "general"
)

func inSet(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

// DeriveSeverity maps an (action, module) pair to a business-risk severity.
// First match wins. Keep this readable — it is the contract a non-technical owner
// implicitly relies on when they filter "show me only the critical things".
func DeriveSeverity(action, module string) string {
	// Destructive actions on a core business entity are the highest concern — a
	// deleted product, voided sale, or cancelled invoice can directly cost money
	// (spec §10: "Delete Product → Critical").
	coreEntity := inSet([]string{
		"product", "sale", "invoice", "stock", "purchasing", "settings",
		"customer", "promotion", "warehouse", "location", "document",
	}, module)

	switch {
	case inSet([]string{"delete", "void", "cancel"}, action) && coreEntity:
		return SeverityCritical
	case module == "settings" && inSet([]string{"manage-bank-accounts", "manage-members"}, action):
		return SeverityCritical
	case inSet([]string{"adjust", "transfer", "receive", "pay", "convert"}, action):
		return SeverityHigh
	case action == "delete":
		return SeverityHigh // delete of a non-core / unknown module
	case action == "update" && inSet([]string{"settings", "product", "purchasing", "promotion"}, module):
		return SeverityHigh // e.g. a price change (spec §10: "Price Change → High")
	case action == "update":
		return SeverityMedium // e.g. editing a minor record (spec §10: "Category Edit → Medium")
	case action == "create" && inSet([]string{"sale", "purchasing", "customer"}, module):
		return SeverityMedium
	default:
		return SeverityNormal
	}
}

// DeriveCategory groups a module (with a light action nuance for security) into one
// of the spec's business categories.
func DeriveCategory(action, module string) string {
	switch module {
	case "product", "stock", "warehouse", "location":
		return CategoryInventory
	case "sale", "parked-bill", "invoice", "document":
		return CategorySales
	case "purchasing":
		return CategoryPurchasing
	case "customer":
		return CategoryCustomer
	case "promotion":
		return CategoryPromotion
	case "expense", "finance":
		return CategoryFinance
	case "settings":
		if inSet([]string{"manage-bank-accounts", "manage-members"}, action) {
			return CategorySecurity
		}
		return CategorySettings
	default:
		return CategoryGeneral
	}
}

// knownModules / knownActions enumerate every canonical value the activity
// middleware can emit (see internal/middleware/activity_log.go). They exist only so
// SeverityPairs / CategoryPairs can translate a derived filter back into concrete
// (module, action) predicates without duplicating the mapping logic.
var knownModules = []string{
	"product", "sale", "parked-bill", "document", "invoice", "customer",
	"warehouse", "purchasing", "stock", "location", "settings", "promotion",
	"expense", "finance", "system",
}

var knownActions = []string{
	"create", "update", "delete", "pay", "cancel", "void", "convert", "adjust",
	"transfer", "receive", "print", "manage-members", "manage-bank-accounts", "get",
}

// SeverityPairs returns every (module, action) tuple whose derived severity equals
// sev. Generated from DeriveSeverity so the filter and the badge can never drift.
func SeverityPairs(sev string) [][2]string {
	return pairsWhere(func(action, module string) bool {
		return DeriveSeverity(action, module) == sev
	})
}

// CategoryPairs returns every (module, action) tuple whose derived category equals
// cat. Same drift-free guarantee as SeverityPairs.
func CategoryPairs(cat string) [][2]string {
	return pairsWhere(func(action, module string) bool {
		return DeriveCategory(action, module) == cat
	})
}

func pairsWhere(match func(action, module string) bool) [][2]string {
	pairs := make([][2]string, 0, 16)
	for _, m := range knownModules {
		for _, a := range knownActions {
			if match(a, m) {
				pairs = append(pairs, [2]string{m, a})
			}
		}
	}
	return pairs
}
