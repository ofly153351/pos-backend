package promotion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"pos-backend/internal/modules/auth"

	"github.com/gofiber/fiber/v2"
)

type validationRepository struct{ writes int }

func (r *validationRepository) List(context.Context, string) ([]json.RawMessage, error) {
	return nil, nil
}
func (r *validationRepository) GetData(context.Context, string, string) (string, error) {
	return `{}`, nil
}
func (r *validationRepository) Insert(context.Context, PromotionRow) error { r.writes++; return nil }
func (r *validationRepository) Update(context.Context, string, string, map[string]any) (int64, error) {
	r.writes++
	return 1, nil
}
func (r *validationRepository) Delete(context.Context, string, string) (int64, error) { return 1, nil }

func TestPromotionValidPercentageBoundariesPreserveStatus(t *testing.T) {
	for _, percent := range []float64{0, 10.5, 100} {
		for _, status := range []string{"", "draft", "active"} {
			repo := &validationRepository{}
			body := fmt.Sprintf(`{"type":"percentage","percentOff":%v,"status":%q,"scopeIds":[]}`, percent, status)
			result, err := NewService(repo).Create(context.Background(), auth.Claims{}, "store-a", []byte(body))
			if err != nil {
				t.Fatal(err)
			}
			var campaign struct {
				Status     string  `json:"status"`
				PercentOff float64 `json:"percentOff"`
			}
			if err := json.Unmarshal(result, &campaign); err != nil {
				t.Fatal(err)
			}
			want := status
			if want == "" {
				want = "draft"
			}
			if campaign.Status != want || campaign.PercentOff != percent || repo.writes != 1 {
				t.Fatalf("unexpected campaign=%s writes=%d", result, repo.writes)
			}
		}
	}
}

func TestPromotionAPIRejectsInvalidDiscounts(t *testing.T) {
	for _, method := range []string{"POST", "PUT"} {
		for _, typ := range []string{"percentage", "happy_hour"} {
			for _, value := range []string{"150", "-1", "100.01", `"150"`, "null"} {
				t.Run(fmt.Sprintf("%s/%s/%s", method, typ, value), func(t *testing.T) {
					repo := &validationRepository{}
					handler := NewHandler(NewService(repo))
					app := fiber.New()
					app.Post("/stores/:storeID/promotions/:promotionID", handler.Create)
					app.Put("/stores/:storeID/promotions/:promotionID", handler.Update)
					body := fmt.Sprintf(`{"type":%q,"percentOff":%s,"status":"active"}`, typ, value)
					req := httptest.NewRequest(method, "/stores/store-a/promotions/promo-a", strings.NewReader(body))
					req.Header.Set("Content-Type", "application/json")
					resp, err := app.Test(req)
					if err != nil {
						t.Fatal(err)
					}
					defer resp.Body.Close()
					if resp.StatusCode != 422 {
						t.Errorf("status=%d, want 422 for %s", resp.StatusCode, body)
					}
					if repo.writes != 0 {
						t.Errorf("invalid promotion persisted: writes=%d", repo.writes)
					}
				})
			}
		}
	}
}
