package warehouse_dashboard

import "errors"

var (
	ErrForbidden     = errors.New("user cannot operate this store")
	ErrInvalidPeriod = errors.New("invalid period, accepted values: 7d, 30d, 3m")
)
