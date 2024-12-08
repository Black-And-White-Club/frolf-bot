// round/events.go

package roundevents

import (
	"encoding/json"
	"time"

	apimodels "github.com/Black-And-White-Club/tcr-bot/round/models"
)

// RoundCreateEvent is triggered when a user creates a new round.// RoundCreateEvent is triggered when a user creates a new round.
type RoundCreateEvent struct {
	DiscordID      string    `json:"discord_id"`
	Date           time.Time `json:"date"`
	Course         string    `json:"course"`
	InitialPlayers []string  `json:"initial_players"`
	Time           string    `json:"time"`
}

// GetDate implements round.RoundCreateEvent.
func (e *RoundCreateEvent) GetDate() time.Time {
	panic("unimplemented")
}

// GetInitialPlayers implements round.RoundCreateEvent.
func (e *RoundCreateEvent) GetInitialPlayers() []string {
	panic("unimplemented")
}

// GetTime implements round.RoundCreateEvent.
func (e *RoundCreateEvent) GetTime() string {
	panic("unimplemented")
}

// Topic returns the topic name for the RoundCreateEvent.
func (e RoundCreateEvent) Topic() string {
	return "round_create"
}
func (e RoundCreateEvent) GetDiscordID() string {
	return e.DiscordID
}

// RoundUpdatedEvent is triggered when a round is updated.
type RoundUpdatedEvent struct {
	RoundID  int64     `json:"round_id"`
	Title    string    `json:"title"`
	Location string    `json:"location"`
	Date     time.Time `json:"date"`
	Time     string    `json:"time"`
}

// Topic returns the topic name for the RoundUpdatedEvent.
func (e RoundUpdatedEvent) Topic() string {
	return "round_updated"
}

// RoundDeletedEvent is triggered when a round is deleted.
type RoundDeletedEvent struct {
	RoundID int64 `json:"round_id"`
}

// Topic returns the topic name for the RoundDeletedEvent.
func (e RoundDeletedEvent) Topic() string {
	return "round_deleted"
}

// PlayerAddedToRoundEvent is triggered when a player is added to a round.
type PlayerAddedToRoundEvent struct {
	RoundID   int64              `json:"round_id"`
	DiscordID string             `json:"discord_id"`
	Response  apimodels.Response `json:"response"`
}

// Topic returns the topic name for the PlayerAddedToRoundEvent.
func (e PlayerAddedToRoundEvent) Topic() string {
	return "player_added_to_round"
}

// PlayerRemovedFromRoundEvent is triggered when a player is removed from a round.
type PlayerRemovedFromRoundEvent struct {
	RoundID   int64  `json:"round_id"`
	DiscordID string `json:"discord_id"`
}

// Topic returns the topic name for the PlayerRemovedFromRoundEvent.
func (e PlayerRemovedFromRoundEvent) Topic() string {
	return "player_removed_from_round"
}

// ScoreSubmittedEvent is triggered when a player submits their score for a round.
type ScoreSubmittedEvent struct {
	RoundID   int64  `json:"round_id"`
	DiscordID string `json:"discord_id"`
	Score     int    `json:"score"`
}

// Topic returns the topic name for the ScoreSubmittedEvent.
func (e ScoreSubmittedEvent) Topic() string {
	return "score_submitted"
}

// TagNumberRequestedEvent is triggered when a player is added to a round
// and their tag number is needed.
type TagNumberRequestedEvent struct {
	DiscordID string `json:"discord_id"`
	RoundID   int64  `json:"round_id"`
}

// Topic returns the topic name for the TagNumberRequestedEvent.
func (e TagNumberRequestedEvent) Topic() string {
	return "tag_number_requested"
}

// TagNumberRetrievedEvent is triggered when a player's tag number is retrieved.
type TagNumberRetrievedEvent struct {
	DiscordID string `json:"discord_id"`
	RoundID   int64  `json:"round_id"`
	TagNumber int    `json:"tag_number"`
}

// Topic returns the topic name for the TagNumberRetrievedEvent.
func (e TagNumberRetrievedEvent) Topic() string {
	return "tag_number_retrieved"
}

// Marshal marshals the RoundFinalizedEvent into a JSON byte array.
func (e RoundFinalizedEvent) Marshal() []byte {
	data, _ := json.Marshal(e)
	return data
}

// RoundFinalizedEvent is triggered when a round is finalized.
type RoundFinalizedEvent struct {
	RoundID      int64                        `json:"round_id"`
	Participants []apimodels.ParticipantScore `json:"participants"` // Corrected to a slice
}

// Topic returns the topic name for the RoundFinalizedEvent.
func (e RoundFinalizedEvent) Topic() string {
	return "round_finalized"
}

// RoundStartedEvent is triggered when a round is started.
type RoundStartedEvent struct {
	RoundID int64 `json:"round_id"`
}

// RoundStartingOneHourEvent is triggered one hour before a round starts.
type RoundStartingOneHourEvent struct {
	RoundID int64 `json:"round_id"`
}

// Topic returns the topic name for the RoundStartingOneHourEvent.
func (e RoundStartingOneHourEvent) Topic() string {
	return "round_starting_one_hour"
}

// RoundStartingThirtyMinutesEvent is triggered 30 minutes before a round starts.
type RoundStartingThirtyMinutesEvent struct {
	RoundID int64 `json:"round_id"`
}

// Topic returns the topic name for the RoundStartingThirtyMinutesEvent.
func (e RoundStartingThirtyMinutesEvent) Topic() string {
	return "round_starting_thirty_minutes"
}

// Topic returns the topic name for the RoundStartedEvent.
func (e RoundStartedEvent) Topic() string {
	return "round_started"
}

// GetRoundID implements the ScoreSubmissionEvent interface.
func (e ScoreSubmittedEvent) GetRoundID() int64 {
	return e.RoundID
}

// GetDiscordID implements the ScoreSubmissionEvent interface.
func (e ScoreSubmittedEvent) GetDiscordID() string {
	return e.DiscordID
}

// GetScore implements the ScoreSubmissionEvent interface.
func (e ScoreSubmittedEvent) GetScore() int {
	return e.Score
}

// GetRoundID implements the RoundStartedEvent interface.
func (e RoundStartedEvent) GetRoundID() int64 {
	return e.RoundID
}

// GetRoundID implements the RoundStartingOneHourEvent interface.
func (e RoundStartingOneHourEvent) GetRoundID() int64 {
	return e.RoundID
}

// GetRoundID implements the RoundStartingThirtyMinutesEvent interface.
func (e RoundStartingThirtyMinutesEvent) GetRoundID() int64 {
	return e.RoundID
}

// GetRoundID implements the RoundUpdatedEvent interface.
func (e RoundUpdatedEvent) GetRoundID() int64 {
	return e.RoundID
}

// GetTitle implements the RoundUpdatedEvent interface.
func (e RoundUpdatedEvent) GetTitle() string {
	return e.Title
}

// GetLocation implements the RoundUpdatedEvent interface.
func (e RoundUpdatedEvent) GetLocation() string {
	return e.Location
}

// GetDate implements the RoundUpdatedEvent interface.
func (e RoundUpdatedEvent) GetDate() time.Time {
	return e.Date
}

// GetTime implements the RoundUpdatedEvent interface.
func (e RoundUpdatedEvent) GetTime() string {
	return e.Time
}

// GetRoundID implements the RoundDeletedEvent interface.
func (e RoundDeletedEvent) GetRoundID() int64 {
	return e.RoundID
}

// GetRoundID implements the RoundFinalizedEvent interface.
func (e RoundFinalizedEvent) GetRoundID() int64 {
	return e.RoundID
}

// GetParticipants implements the RoundFinalizedEvent interface.
func (e RoundFinalizedEvent) GetParticipants() []apimodels.ParticipantScore {
	return e.Participants
}

// GetCourse implements the round.RoundCreateEvent interface.
func (e RoundCreateEvent) GetCourse() string {
	return e.Course
}
