package leaderboarddomain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// RoundInput represents the raw input for a round that needs to be hashed for idempotency.
type RoundInput struct {
	MemberID   string
	FinishRank int
}

// ComputeProcessingHash generates a deterministic hash from round input data.
// This hash is used to detect whether a round has already been processed with the same data,
// or if it needs recalculation (same round_id but different data).
func ComputeProcessingHash(inputs []RoundInput) string {
	// Sort for determinism
	sorted := make([]RoundInput, len(inputs))
	copy(sorted, inputs)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].FinishRank != sorted[j].FinishRank {
			return sorted[i].FinishRank < sorted[j].FinishRank
		}
		return sorted[i].MemberID < sorted[j].MemberID
	})

	var sb strings.Builder
	for _, inp := range sorted {
		fmt.Fprintf(&sb, "%s:%d;", inp.MemberID, inp.FinishRank)
	}

	hash := sha256.Sum256([]byte(sb.String()))
	return hex.EncodeToString(hash[:])
}
