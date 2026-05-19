package stock_movement

import (
	"context"
	"fmt"
	"strings"
	"time"

	"pos-backend/internal/modules/auth"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (s Service) AddStock(ctx context.Context, actor auth.Claims, storeID string, input AddStockRequest) (AdditionResult, error) {
	if strings.TrimSpace(storeID) == "" {
		return AdditionResult{}, fmt.Errorf("storeID is required")
	}

	if len(input.Items) == 0 {
		return AdditionResult{}, ErrStockNoItems
	}

	for _, item := range input.Items {
		if item.Quantity <= 0 {
			return AdditionResult{}, ErrStockBadQty
		}
	}

	now := time.Now().UTC()
	var movements []StockMovement

	for _, item := range input.Items {
		// Verify product exists in store
		if _, err := s.repo.GetProduct(ctx, storeID, item.ProductID); err != nil {
			return AdditionResult{}, err
		}

		mg := StockMovement{
			ID:             newID(),
			StoreID:        storeID,
			ProductID:      item.ProductID,
			QuantityChange: item.Quantity,
			Type:           "stock_in",
			Note:           strings.TrimSpace(item.Note),
			CreatedBy:      actor.UserID,
			CreatedAt:      now,
		}

		created, err := s.repo.Create(ctx, mg)
		if err != nil {
			return AdditionResult{}, err
		}

		// Update product quantity
		if err := s.repo.UpdateProductQuantity(ctx, item.ProductID, item.Quantity); err != nil {
			return AdditionResult{}, err
		}

		movements = append(movements, created)
	}

	return AdditionResult{Movements: movements}, nil
}

func (s Service) ListMovements(ctx context.Context, actor auth.Claims, storeID string, q ListMovementsQuery) (MovementResponse, error) {
	if strings.TrimSpace(storeID) == "" {
		return MovementResponse{}, fmt.Errorf("storeID is required")
	}

	// Ensure store access is checked via handler
	return s.repo.ListByStore(ctx, storeID, q)
}
