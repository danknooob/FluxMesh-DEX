package service

import "errors"

var (
	ErrMarketDisabled      = errors.New("market is disabled")
	ErrMarketNotFound      = errors.New("market not found")
	ErrInvalidSide         = errors.New("invalid side: must be buy or sell")
	ErrOrderNotFound       = errors.New("order not found")
	ErrOrderNotCancellable = errors.New("order cannot be cancelled")
)
