package promotion

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"pos-backend/internal/modules/auth"

	"github.com/gofiber/fiber/v2"
)

func TestPromotionAPIRejectsCaseVariantParameters(t *testing.T) {
	for _, method := range []string{"POST", "PUT"} {
		for _, body := range []string{
			`{"type":"percentage","status":"active","percentOff":10,"percentoff":150}`,
			`{"type":"fixed_price","status":"active","fixedPrice":50,"fixedprice":-1}`,
			`{"type":"percentage","status":"paused","Status":"active","percentOff":150}`,
		} {
			t.Run(method+body, func(t *testing.T) {
				repo := &validationRepository{}
				handler := NewHandler(NewService(repo))
				app := fiber.New()
				app.Post("/stores/:storeID/promotions/:promotionID", handler.Create)
				app.Put("/stores/:storeID/promotions/:promotionID", handler.Update)
				req := httptest.NewRequest(method, "/stores/store-a/promotions/promo-a", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				res, err := app.Test(req)
				if err != nil {
					t.Fatal(err)
				}
				defer res.Body.Close()
				if res.StatusCode != 422 || repo.writes != 0 {
					t.Fatalf("ambiguous parameters accepted: status=%d writes=%d", res.StatusCode, repo.writes)
				}
			})
		}
	}
}

func TestPromotionCanPauseInvalidHistoricalDiscount(t *testing.T) {
	repo := &validationRepository{}
	_, err := NewService(repo).Update(context.Background(), auth.Claims{}, "store-a", "promo-a", []byte(`{"type":"percentage","status":"paused","percentOff":150}`))
	if err != nil || repo.writes != 1 {
		t.Fatalf("must allow deactivation without repairing old discount: err=%v writes=%d", err, repo.writes)
	}
	_, err = NewService(repo).Update(context.Background(), auth.Claims{}, "store-a", "promo-a", []byte(`{"type":"percentage","status":"active","percentOff":150}`))
	if err == nil || repo.writes != 1 {
		t.Fatalf("invalid campaign must not reactivate: err=%v writes=%d", err, repo.writes)
	}
}

func TestPromotionIncompleteDraftCannotActivate(t *testing.T) {
	for _, status := range []string{"draft", "active"} {
		t.Run(status, func(t *testing.T) {
			repo := &validationRepository{}
			body := []byte(`{"name":"Work in progress","type":"percentage","status":"` + status + `"}`)
			_, err := NewService(repo).Create(context.Background(), auth.Claims{}, "store-a", body)
			if status == "draft" {
				if err != nil || repo.writes != 1 {
					t.Fatalf("incomplete draft should be saved: err=%v writes=%d", err, repo.writes)
				}
			} else if err == nil || repo.writes != 0 {
				t.Fatalf("incomplete active campaign accepted: err=%v writes=%d", err, repo.writes)
			}
		})
	}
}
