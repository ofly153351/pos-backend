package middleware

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"

	"pos-backend/internal/modules/activity_log"
	"pos-backend/internal/platform/activitycapture"
)

// moduleMap maps URL segment to canonical module name.
var moduleMap = map[string]string{
	"products":          "product",
	"product-types":     "product",
	"product-units":     "product",
	"product-brands":    "product",
	"sales":             "sale",
	"parked-bills":      "parked-bill",
	"documents":         "document",
	"invoices":          "invoice",
	"customers":         "customer",
	"warehouses":        "warehouse",
	"warehouse-receipts": "warehouse",
	"purchasing":        "purchasing",
	"purchase-orders":   "purchasing",
	"stock-movements":   "stock",
	"stock":             "stock",
	"promotions":        "promotion",
	"locations":         "location",
	"vat":               "settings",
	"bank-accounts":     "settings",
	"receipt-settings":  "settings",
	"stores":            "settings",
	"members":           "settings",
}

// actionMap maps sub-path segment (after the resource ID) to canonical action.
var actionMap = map[string]string{
	"pay":        "pay",
	"cancel":     "cancel",
	"void":       "void",
	"convert":    "convert",
	"adjust":     "adjust",
	"transfer":   "transfer",
	"receive":    "receive",
	"print":      "print",
	"members":    "manage-members",
	"bank-accounts": "manage-bank-accounts",
}

func parseModuleAction(method, rawPath string) (module, action, resourceID string) {
	// rawPath example: /api/v1/stores/str-00001234/products/pd-00001234
	// Strip common API prefixes
	path := rawPath
	for _, prefix := range []string{"/api/v1/stores/", "/api/stores/"} {
		if strings.HasPrefix(path, prefix) {
			path = strings.TrimPrefix(path, prefix)
			break
		}
	}
	// path is now: {storeID}/products/pd-00001234
	parts := strings.SplitN(path, "/", 4)
	// parts[0]=storeID, parts[1]=module-segment, parts[2]=resourceID (opt), parts[3]=subaction (opt)

	if len(parts) < 2 {
		return "system", strings.ToLower(method), ""
	}

	seg := parts[1]
	module = moduleMap[seg]
	if module == "" {
		module = seg
	}

	switch {
	case len(parts) >= 4:
		// e.g. documents/{id}/pay
		sub := parts[3]
		if a, ok := actionMap[sub]; ok {
			action = a
		} else {
			action = sub
		}
		resourceID = parts[2]
	case len(parts) == 3:
		// e.g. products/{id}
		resourceID = parts[2]
		switch method {
		case "PUT", "PATCH":
			action = "update"
		case "DELETE":
			action = "delete"
		default:
			action = strings.ToLower(method)
		}
	default:
		// e.g. POST /products
		switch method {
		case "POST":
			action = "create"
		default:
			action = strings.ToLower(method)
		}
	}

	return module, action, resourceID
}

// ActivityLog returns a Fiber middleware that asynchronously logs every
// successful mutating request (POST/PUT/PATCH/DELETE) to the activity_logs table.
func ActivityLog(svc activity_log.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Install a change recorder so service-layer Update methods can capture a
		// {before, after} field diff that we attach to the audit row below.
		recorder := activitycapture.New()
		c.SetUserContext(activitycapture.WithRecorder(c.UserContext(), recorder))

		err := c.Next()

		method := c.Method()
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			return err
		}

		// Only log successful responses
		status := c.Response().StatusCode()
		if status >= 400 {
			return err
		}

		storeID := c.Params("storeID")
		if storeID == "" {
			return err
		}

		claims := ClaimsFromContext(c)
		path := c.Path()
		module, action, resourceID := parseModuleAction(method, path)
		changes := recorder.JSON() // nil unless a service recorded a diff

		go svc.Log(context.Background(), activity_log.LogRequest{
			StoreID:    storeID,
			UserID:     claims.UserID,
			UserName:   claims.Name,
			Action:     action,
			Module:     module,
			ResourceID: resourceID,
			Method:     method,
			Path:       path,
			IPAddress:  c.IP(),
			Changes:    activity_log.JSONText(changes),
		})

		return err
	}
}
