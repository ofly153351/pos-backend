package usersettings

import (
	"context"
	"errors"
	"strings"
)

var ErrUserIDRequired = errors.New("user id is required")

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

// GetCardSettings returns the user's saved settings, or defaults when unset.
func (s Service) GetCardSettings(ctx context.Context, userID string) (CardSettings, error) {
	if strings.TrimSpace(userID) == "" {
		return CardSettings{}, ErrUserIDRequired
	}
	cs, ok, err := s.repo.GetCardSettings(ctx, userID)
	if err != nil {
		return CardSettings{}, err
	}
	if !ok {
		return DefaultCardSettings(), nil
	}
	return cs.normalize(), nil
}

// SaveCardSettings validates + persists the settings for the user.
func (s Service) SaveCardSettings(ctx context.Context, userID string, in CardSettings) (CardSettings, error) {
	if strings.TrimSpace(userID) == "" {
		return CardSettings{}, ErrUserIDRequired
	}
	normalized := in.normalize()
	if err := s.repo.SaveCardSettings(ctx, userID, normalized); err != nil {
		return CardSettings{}, err
	}
	return normalized, nil
}
