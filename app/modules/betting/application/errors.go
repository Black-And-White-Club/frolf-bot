package bettingservice

import "errors"

var (
	ErrMembershipRequired       = errors.New("betting membership required")
	ErrFeatureDisabled          = errors.New("betting feature disabled")
	ErrFeatureFrozen            = errors.New("betting feature frozen")
	ErrAdminRequired            = errors.New("betting admin role required")
	ErrTargetMemberNotFound     = errors.New("betting target member not found")
	ErrAdjustmentAmountInvalid  = errors.New("betting adjustment amount must be non-zero")
	ErrAdjustmentReasonRequired = errors.New("betting adjustment reason required")
	ErrReasonTooLong            = errors.New("adjustment reason exceeds maximum length")
	ErrNoEligibleRound          = errors.New("betting no eligible round")
	ErrBetStakeInvalid          = errors.New("betting stake must be positive")
	ErrSelectionInvalid         = errors.New("betting selection invalid")
	ErrInsufficientBalance      = errors.New("betting insufficient balance")
	ErrMarketLocked             = errors.New("betting market locked")
	ErrMarketNotFound           = errors.New("betting market not found")
	ErrInvalidMarketAction      = errors.New("betting invalid market action")
	ErrRoundNotFinalized        = errors.New("betting round not finalized")
	ErrSelfBetProhibited        = errors.New("betting cannot bet on yourself in this market")
	ErrInvalidMarketType        = errors.New("betting invalid market type")
)
