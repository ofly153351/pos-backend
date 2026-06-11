package finance

import "errors"

var (
	ErrInvalidPeriod        = errors.New("invalid period, allowed values: 7d, 30d, 90d")
	ErrInvalidTimeRange     = errors.New("invalid time range")
	ErrForbiddenStoreAccess = errors.New("user cannot operate pos for this store")
)
