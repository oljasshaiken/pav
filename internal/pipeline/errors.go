package pipeline

import "errors"

var (
	ErrClaimNotFound  = errors.New("claim not found")
	ErrConfigNotFound = errors.New("config not found")
)
