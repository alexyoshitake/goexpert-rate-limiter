package ratelimiter

import "errors"

var (
	ErrStorageUnavailable = errors.New("armazenamento do rate limiter indisponível")
	ErrInvalidLimit       = errors.New("o limite do rate limiter deve ser maior que zero")
	ErrInvalidBlockTime   = errors.New("o tempo de bloqueio do rate limiter deve ser maior que zero")
)
