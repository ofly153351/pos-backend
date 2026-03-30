package dashboard

import "errors"

var (
	ErrInvalidPeriod        = errors.New("invalid period, allowed values: today, 7d, 30d")
	ErrInvalidTimeRange     = errors.New("invalid time range")
	ErrInvalidLimit         = errors.New("invalid limit")
	ErrForbiddenStoreAccess = errors.New("user cannot operate pos for this store")
)
