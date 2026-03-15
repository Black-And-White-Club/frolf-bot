package bettingservice

const (
	defaultSeasonID         = "default"
	defaultSeasonName       = "Default Season"
	walletHistorySize       = 20
	adminMarketListSize     = 25
	winnerMarketType        = "round_winner"
	placement2ndMarketType  = "placement_2nd"
	placement3rdMarketType  = "placement_3rd"
	placementLastMarketType = "placement_last"
	overUnderMarketType     = "over_under"
	openMarketStatus        = "open"
	lockedMarketStatus      = "locked"
	suspendedMarketStatus   = "suspended"
	settledMarketStatus     = "settled"
	voidedMarketStatus      = "voided"
	acceptedBetStatus       = "accepted"
	wonBetStatus            = "won"
	lostBetStatus           = "lost"
	voidedBetStatus         = "voided"
	stakeReservedEntry      = "stake_reserved"
	marketSettlementEntry   = "market_settlement"
	marketRefundEntry       = "market_refund"
	marketCorrectionEntry   = "market_correction"
	marketVigMultiplier     = 1.05
	minMarketProbability    = 0.01
	maxMarketProbability    = 0.95
	minDecimalOddsCents     = 105
	winnerMarketScale       = 250.0
	adminActionVoid         = "void"
	adminActionResettle     = "resettle"
)
