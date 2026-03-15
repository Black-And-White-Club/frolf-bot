package bettingdb

import "errors"

var (
	ErrNotFound                  = errors.New("betting record not found")
	ErrNoRowsAffected            = errors.New("no rows affected")
	ErrSettlementVersionConflict = errors.New("settlement version conflict: market was concurrently settled")
)
