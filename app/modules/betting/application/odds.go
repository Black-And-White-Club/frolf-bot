package bettingservice

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

const (
	// historyWindow is how far back we look for historical rounds.
	historyWindow = 90 * 24 * time.Hour

	// monteCarloN is the number of simulation iterations for win probability.
	monteCarloN = 500

	// baselineMu is the assumed mean skill for players with no recorded history.
	// Lower scores are better in disc golf, so 0.5 maps to the middle of the
	// normalised [0,1] difficulty scale we use internally.
	baselineMu = 0.5

	// baselineSigma is the uncertainty assigned to players with no history.
	baselineSigma = 0.35

	// fieldSizeCalibrationRate widens variance for larger fields so heavy
	// favourites aren't priced too short.
	fieldSizeCalibrationRate = 0.02

	// decayFactor reduces the weight of older rounds exponentially.
	decayFactor = 0.85
)

// oddsEngine computes Bayesian win probabilities for a set of participants.
type oddsEngine struct {
	roundRepo       roundRepository
	leaderboardRepo leaderboardRepository
}

func newOddsEngine(roundRepo roundRepository, leaderboardRepo leaderboardRepository) *oddsEngine {
	return &oddsEngine{roundRepo: roundRepo, leaderboardRepo: leaderboardRepo}
}

// playerRating holds the estimated skill distribution for a single player.
type playerRating struct {
	mu    float64 // normalised mean skill (higher = better)
	sigma float64 // uncertainty (standard deviation)
}

// simulationResult holds the aggregate counts from monteCarloN iterations.
type simulationResult struct {
	// winCounts[i] = number of iterations player i won.
	winCounts []int
	// placementCounts[i][rank] = number of iterations player i finished at rank
	// (0-indexed: rank 0 = 1st place, rank 1 = 2nd place, …).
	placementCounts [][]int
	// scoreSamples[i][iter] = the normalised sampled score for player i in
	// iteration iter. Used for O/U probability derivation.
	scoreSamples [][]float64
}

// simulateFull runs monteCarloN iterations and collects win counts, per-rank
// placement counts, and raw score samples for every player.
func simulateFull(ratings []playerRating) simulationResult {
	n := len(ratings)
	res := simulationResult{
		winCounts:       make([]int, n),
		placementCounts: make([][]int, n),
		scoreSamples:    make([][]float64, n),
	}
	for i := range n {
		res.placementCounts[i] = make([]int, n)
		res.scoreSamples[i] = make([]float64, monteCarloN)
	}

	type indexedScore struct {
		idx   int
		score float64
	}

	for iter := range monteCarloN {
		samples := make([]indexedScore, n)
		for i, r := range ratings {
			s := sampleNormal(r.mu, r.sigma)
			samples[i] = indexedScore{idx: i, score: s}
			res.scoreSamples[i][iter] = s
		}

		// Sort descending: highest sampled score = best performance (normalised
		// where higher = better).
		sort.Slice(samples, func(a, b int) bool {
			return samples[a].score > samples[b].score
		})

		// Assign ranks. Handle ties: all players with the same score as the
		// player at position p share that rank.
		rank := 0
		for pos := 0; pos < n; {
			tieScore := samples[pos].score
			tieEnd := pos + 1
			for tieEnd < n && math.Abs(samples[tieEnd].score-tieScore) < 1e-9 {
				tieEnd++
			}
			for k := pos; k < tieEnd; k++ {
				res.placementCounts[samples[k].idx][rank]++
			}
			// Win count: first-place group.
			if rank == 0 {
				res.winCounts[samples[pos].idx]++
			}
			rank += tieEnd - pos
			pos = tieEnd
		}
	}

	return res
}

// priceFromCounts converts a raw count to decimal odds applying vig+clamp.
func priceFromCounts(count, total int) (prob float64, decimalOddsCents int) {
	rawProb := float64(count) / float64(total)
	if rawProb == 0 {
		rawProb = 0.001
	}
	pricedProb := rawProb * marketVigMultiplier
	if pricedProb < minMarketProbability {
		pricedProb = minMarketProbability
	}
	if pricedProb > maxMarketProbability {
		pricedProb = maxMarketProbability
	}
	dc := int(math.Round((1 / pricedProb) * 100))
	if dc < minDecimalOddsCents {
		dc = minDecimalOddsCents
	}
	return rawProb, dc
}

// priceWinnerOptions prices a set of participants using the Bayesian engine and
// returns ordered pricedOptions. This replaces the basic exponential-weight algorithm.
func (e *oddsEngine) priceWinnerOptions(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	participants []targetParticipant,
) ([]pricedOption, error) {
	if len(participants) < 2 {
		return nil, ErrNoEligibleRound
	}

	fieldSize := len(participants)

	historySince := time.Now().Add(-historyWindow)
	history, err := e.roundRepo.GetFinalizedRoundsAfter(ctx, db, guildID, historySince)
	if err != nil {
		history = nil
	}

	observations := buildObservations(history, participants)
	ratings := buildRatings(observations, participants, fieldSize)

	sim := simulateFull(ratings)

	options := make([]pricedOption, 0, fieldSize)
	for i, p := range participants {
		rawProb, dc := priceFromCounts(sim.winCounts[i], monteCarloN)
		options = append(options, pricedOption{
			optionKey:        string(p.participant.UserID),
			memberID:         p.participant.UserID,
			label:            p.label,
			probabilityBps:   int(math.Round(rawProb * 10000)),
			decimalOddsCents: dc,
		})
	}

	sort.Slice(options, func(i, j int) bool {
		if options[i].probabilityBps == options[j].probabilityBps {
			return options[i].label < options[j].label
		}
		return options[i].probabilityBps > options[j].probabilityBps
	})

	return options, nil
}

// pricePlacementOptions prices exact-finish-position markets. position is
// 1-indexed (1=1st, 2=2nd, …). It reuses simulateFull internally.
func (e *oddsEngine) pricePlacementOptions(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	participants []targetParticipant,
	position int, // 1-indexed target rank
) ([]pricedOption, error) {
	if len(participants) < 2 {
		return nil, ErrNoEligibleRound
	}

	fieldSize := len(participants)
	rankIdx := position - 1 // 0-indexed

	historySince := time.Now().Add(-historyWindow)
	history, err := e.roundRepo.GetFinalizedRoundsAfter(ctx, db, guildID, historySince)
	if err != nil {
		history = nil
	}

	observations := buildObservations(history, participants)
	ratings := buildRatings(observations, participants, fieldSize)

	sim := simulateFull(ratings)

	options := make([]pricedOption, 0, fieldSize)
	for i, p := range participants {
		count := 0
		if rankIdx < len(sim.placementCounts[i]) {
			count = sim.placementCounts[i][rankIdx]
		}
		rawProb, dc := priceFromCounts(count, monteCarloN)
		options = append(options, pricedOption{
			optionKey:        string(p.participant.UserID),
			memberID:         p.participant.UserID,
			label:            p.label,
			probabilityBps:   int(math.Round(rawProb * 10000)),
			decimalOddsCents: dc,
		})
	}

	sort.Slice(options, func(i, j int) bool {
		if options[i].probabilityBps == options[j].probabilityBps {
			return options[i].label < options[j].label
		}
		return options[i].probabilityBps > options[j].probabilityBps
	})

	return options, nil
}

// priceOverUnderOptions prices per-player score over/under markets.
// Two options are generated per player: "{discordID}_over" and "{discordID}_under".
// The line is the recency-weighted mean raw score, rounded to the nearest integer.
// P(over) = P(sampled normalised score < mu) from simulation (worse than expected = higher stroke count).
func (e *oddsEngine) priceOverUnderOptions(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	participants []targetParticipant,
) ([]pricedOption, error) {
	if len(participants) < 2 {
		return nil, ErrNoEligibleRound
	}

	fieldSize := len(participants)

	historySince := time.Now().Add(-historyWindow)
	history, err := e.roundRepo.GetFinalizedRoundsAfter(ctx, db, guildID, historySince)
	if err != nil {
		history = nil
	}

	observations := buildObservations(history, participants)
	rawObservations := buildRawObservations(history, participants)
	ratings := buildRatings(observations, participants, fieldSize)

	sim := simulateFull(ratings)

	// Compute field average raw score as fallback for players with no history.
	fieldAvgLine := computeFieldAverageLine(rawObservations)

	options := make([]pricedOption, 0, len(participants)*2)
	for i, p := range participants {
		pid := string(p.participant.UserID)

		line := computePlayerLine(rawObservations[pid], fieldAvgLine)
		lineJSON := fmt.Sprintf(`{"line":%d}`, line)

		// P(over) from MC: fraction of iterations where sampled score < player's mu
		// (lower normalised = worse = more strokes = over the line).
		overCount := 0
		for _, s := range sim.scoreSamples[i] {
			if s < ratings[i].mu {
				overCount++
			}
		}
		underCount := monteCarloN - overCount

		_, overDC := priceFromCounts(overCount, monteCarloN)
		_, underDC := priceFromCounts(underCount, monteCarloN)

		overProb := float64(overCount) / float64(monteCarloN)
		underProb := float64(underCount) / float64(monteCarloN)

		options = append(options,
			pricedOption{
				optionKey:        pid + "_over",
				memberID:         p.participant.UserID,
				label:            fmt.Sprintf("%s Over %d", p.label, line),
				probabilityBps:   int(math.Round(overProb * 10000)),
				decimalOddsCents: overDC,
				metadata:         lineJSON,
			},
			pricedOption{
				optionKey:        pid + "_under",
				memberID:         p.participant.UserID,
				label:            fmt.Sprintf("%s Under %d", p.label, line),
				probabilityBps:   int(math.Round(underProb * 10000)),
				decimalOddsCents: underDC,
				metadata:         lineJSON,
			},
		)
	}

	return options, nil
}

// ---------------------------------------------------------------------------
// Observation / rating helpers
// ---------------------------------------------------------------------------

// observation captures one player's score in one historical round.
type observation struct {
	normalised float64 // 1.0 = best in field, 0.0 = worst
	roundAge   int     // index from most recent (0) → oldest
}

// rawObservation captures a player's un-normalised raw stroke count.
type rawObservation struct {
	score    int
	roundAge int
}

// buildObservations extracts normalised performance metrics for each participant
// from the historical rounds slice. Rounds should be ordered oldest→newest.
func buildObservations(history []*roundtypes.Round, participants []targetParticipant) map[string][]observation {
	obs := make(map[string][]observation)

	participantSet := make(map[string]struct{}, len(participants))
	for _, p := range participants {
		participantSet[string(p.participant.UserID)] = struct{}{}
	}

	for age, round := range reverseRounds(history) {
		if round == nil || len(round.Participants) == 0 {
			continue
		}

		type playerScore struct {
			userID string
			score  int
		}
		var scores []playerScore
		for _, participant := range round.Participants {
			if _, ok := participantSet[string(participant.UserID)]; !ok {
				continue
			}
			if participant.Score == nil || participant.IsDNF {
				continue
			}
			scores = append(scores, playerScore{
				userID: string(participant.UserID),
				score:  int(*participant.Score),
			})
		}
		if len(scores) < 2 {
			continue
		}

		sort.Slice(scores, func(i, j int) bool { return scores[i].score < scores[j].score })

		best := float64(scores[0].score)
		worst := float64(scores[len(scores)-1].score)
		spread := worst - best
		if spread == 0 {
			spread = 1
		}

		for _, ps := range scores {
			normalised := 1.0 - (float64(ps.score)-best)/spread
			obs[ps.userID] = append(obs[ps.userID], observation{
				normalised: normalised,
				roundAge:   age,
			})
		}
	}

	return obs
}

// buildRawObservations collects raw stroke counts (un-normalised) per player.
func buildRawObservations(history []*roundtypes.Round, participants []targetParticipant) map[string][]rawObservation {
	obs := make(map[string][]rawObservation)

	participantSet := make(map[string]struct{}, len(participants))
	for _, p := range participants {
		participantSet[string(p.participant.UserID)] = struct{}{}
	}

	for age, round := range reverseRounds(history) {
		if round == nil {
			continue
		}
		for _, participant := range round.Participants {
			if _, ok := participantSet[string(participant.UserID)]; !ok {
				continue
			}
			if participant.Score == nil || participant.IsDNF {
				continue
			}
			pid := string(participant.UserID)
			obs[pid] = append(obs[pid], rawObservation{
				score:    int(*participant.Score),
				roundAge: age,
			})
		}
	}

	return obs
}

// computePlayerLine returns the recency-weighted mean raw score for a player,
// rounded to the nearest integer. Falls back to fieldAvgLine if no history.
func computePlayerLine(obs []rawObservation, fieldAvgLine int) int {
	if len(obs) == 0 {
		return fieldAvgLine
	}
	var weightedSum, totalWeight float64
	for _, o := range obs {
		w := math.Pow(decayFactor, float64(o.roundAge))
		weightedSum += w * float64(o.score)
		totalWeight += w
	}
	if totalWeight == 0 {
		return fieldAvgLine
	}
	return int(math.Round(weightedSum / totalWeight))
}

// computeFieldAverageLine computes the simple unweighted mean raw score across
// all players who have history, used as a fallback line for history-less players.
func computeFieldAverageLine(rawObs map[string][]rawObservation) int {
	var total float64
	var count int
	for _, obs := range rawObs {
		for _, o := range obs {
			total += float64(o.score)
			count++
		}
	}
	if count == 0 {
		return 54 // sensible disc golf fallback (par 54 = typical 18-hole course)
	}
	return int(math.Round(total / float64(count)))
}

// buildRatings constructs per-player skill ratings from observations and applies
// field-size calibration.
func buildRatings(observations map[string][]observation, participants []targetParticipant, fieldSize int) []playerRating {
	ratings := make([]playerRating, len(participants))
	for i, p := range participants {
		ratings[i] = computeRating(observations[string(p.participant.UserID)], fieldSize)
	}
	calibration := 1.0 + fieldSizeCalibrationRate*float64(fieldSize-2)
	for i := range ratings {
		ratings[i].sigma *= calibration
	}
	return ratings
}

// reverseRounds returns a copy of the slice in reverse order (newest first)
// together with an age index (0 = most recent).
func reverseRounds(rounds []*roundtypes.Round) []*roundtypes.Round {
	n := len(rounds)
	out := make([]*roundtypes.Round, n)
	for i, r := range rounds {
		out[n-1-i] = r
	}
	return out
}

// computeRating estimates a player's Bayesian skill distribution from their
// observation history. Returns baseline if no observations are available.
func computeRating(obs []observation, fieldSize int) playerRating {
	if len(obs) == 0 {
		return playerRating{mu: baselineMu, sigma: baselineSigma}
	}

	var weightedSum, totalWeight float64
	for _, o := range obs {
		w := math.Pow(decayFactor, float64(o.roundAge))
		weightedSum += w * o.normalised
		totalWeight += w
	}
	if totalWeight == 0 {
		return playerRating{mu: baselineMu, sigma: baselineSigma}
	}
	mu := weightedSum / totalWeight

	var varianceSum float64
	for _, o := range obs {
		w := math.Pow(decayFactor, float64(o.roundAge))
		diff := o.normalised - mu
		varianceSum += w * diff * diff
	}
	variance := varianceSum / totalWeight
	sigma := math.Sqrt(variance)

	minSigma := baselineSigma / math.Sqrt(float64(len(obs)+1))
	if sigma < minSigma {
		sigma = minSigma
	}

	_ = fieldSize
	return playerRating{mu: mu, sigma: sigma}
}

// sampleNormal samples from N(mu, sigma) using the Box-Muller transform.
func sampleNormal(mu, sigma float64) float64 {
	if sigma <= 0 {
		return mu
	}
	u1 := rand.Float64()
	u2 := rand.Float64()
	if u1 == 0 {
		u1 = 1e-10
	}
	z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	return mu + sigma*z
}

// priceWinnerOptionsFromStandings is the legacy fallback used when the
// round history query is unavailable. It mirrors the original point-weight
// algorithm. The oddsEngine delegates to this only if the Bayesian path fails.
func priceWinnerOptionsFromStandings(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	participants []targetParticipant,
	lr leaderboardRepository,
) ([]pricedOption, error) {
	type candidate struct {
		participant targetParticipant
		points      int
		weight      float64
	}

	candidates := make([]candidate, 0, len(participants))
	maxPoints := 0
	for _, participant := range participants {
		points := 0
		standing, err := lr.GetSeasonStanding(ctx, db, string(guildID), participant.participant.UserID)
		if err != nil {
			return nil, err
		}
		if standing != nil {
			points = standing.TotalPoints
		}
		if points > maxPoints {
			maxPoints = points
		}
		candidates = append(candidates, candidate{participant: participant, points: points})
	}

	totalWeight := 0.0
	for idx := range candidates {
		weight := math.Exp(float64(candidates[idx].points-maxPoints) / winnerMarketScale)
		candidates[idx].weight = weight
		totalWeight += weight
	}
	if totalWeight == 0 {
		totalWeight = float64(len(candidates))
		for idx := range candidates {
			candidates[idx].weight = 1
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].points == candidates[j].points {
			return candidates[i].participant.label < candidates[j].participant.label
		}
		return candidates[i].points > candidates[j].points
	})

	options := make([]pricedOption, 0, len(candidates))
	for _, candidate := range candidates {
		probability := candidate.weight / totalWeight
		pricedProbability := probability * marketVigMultiplier
		if pricedProbability < minMarketProbability {
			pricedProbability = minMarketProbability
		}
		if pricedProbability > maxMarketProbability {
			pricedProbability = maxMarketProbability
		}
		decimalOddsCents := int(math.Round((1 / pricedProbability) * 100))
		if decimalOddsCents < minDecimalOddsCents {
			decimalOddsCents = minDecimalOddsCents
		}
		options = append(options, pricedOption{
			optionKey:        string(candidate.participant.participant.UserID),
			memberID:         candidate.participant.participant.UserID,
			label:            candidate.participant.label,
			probabilityBps:   int(math.Round(probability * 10000)),
			decimalOddsCents: decimalOddsCents,
		})
	}
	return options, nil
}
