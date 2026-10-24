package redis

import "errors"

var (
	ErrClientRequired = errors.New("redis client is required")
	ErrInvalidTTL     = errors.New("TTL must be non-negative")
)
