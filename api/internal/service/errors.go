package service

import "errors"

var (
	ErrMarketDisabled      = errors.New("market is disabled")
	ErrInvalidSide         = errors.New("invalid side: must be buy or sell")
	ErrOrderNotFound       = errors.New("order not found")
	ErrOrderNotCancellable = errors.New("order cannot be cancelled")
)
