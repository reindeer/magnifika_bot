package model

import "errors"

var (
	ErrUnknownCustomer = errors.New("unknown customer")
	ErrUnknownPhone    = errors.New("phone is not registered")
	ErrPhoneChanged    = errors.New("customer phone is no longer registered")
	ErrRateLimited     = errors.New("messenger rate limit")
)
