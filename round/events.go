// round/events.go
package round

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

// PlayerAddedToRoundEventHandler handles PlayerAddedToRoundEvent.
type PlayerAddedToRoundEventHandler interface {
	HandlePlayerAddedToRound(ctx context.Context, msg *message.Message) error
}

// TagNumberRetrievedEventHandler handles TagNumberRetrievedEvent.
type TagNumberRetrievedEventHandler interface {
	HandleTagNumberRetrieved(ctx context.Context, msg *message.Message) error
}

// RoundStartedEventHandler handles RoundStartedEvent.
type RoundStartedEventHandler interface {
	HandleRoundStarted(ctx context.Context, event *RoundStartedEvent) error
}

// RoundStartingOneHourEventHandler handles RoundStartingOneHourEvent.
type RoundStartingOneHourEventHandler interface {
	HandleRoundStartingOneHour(ctx context.Context, event *RoundStartingOneHourEvent) error
}

// RoundStartingThirtyMinutesEventHandler handles RoundStartingThirtyMinutesEvent.
type RoundStartingThirtyMinutesEventHandler interface {
	HandleRoundStartingThirtyMinutes(ctx context.Context, event *RoundStartingThirtyMinutesEvent) error
}

// RoundUpdatedEventHandler handles RoundUpdatedEvent.
type RoundUpdatedEventHandler interface {
	HandleRoundUpdated(ctx context.Context, event *RoundUpdatedEvent) error
}

// RoundDeletedEventHandler handles RoundDeletedEvent.
type RoundDeletedEventHandler interface {
	HandleRoundDeleted(ctx context.Context, event *RoundDeletedEvent) error
}

// RoundFinalizedEventHandler handles RoundFinalizedEvent.
type RoundFinalizedEventHandler interface {
	HandleRoundFinalized(ctx context.Context, event *RoundFinalizedEvent) error
}

// ScoreSubmittedEventHandler handles ScoreSubmittedEvent.
type ScoreSubmittedEventHandler interface {
	HandleScoreSubmitted(ctx context.Context, event *ScoreSubmittedEvent) error
}

// RoundCreateEventHandler handles RoundCreateEvent.
type RoundCreateEventHandler interface {
	HandleRoundCreate(ctx context.Context, event *RoundCreateEvent) error
}

// RoundCreateEvent is triggered when a user creates a new round.
type RoundCreateEvent struct {
	UserID         string    `json:"user_id"`
	Date           time.Time `json:"date"`
	Course         string    `json:"course"`
	InitialPlayers []string  `json:"initial_players"`
	Time           string    `json:"time"`
}

// Topic returns the topic name for the RoundCreateEvent.
func (e RoundCreateEvent) Topic() string {
	return "round_create"
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
	RoundID  int64    `json:"round_id"`
	UserID   string   `json:"user_id"`
	Response Response `json:"response"` // Assuming you have a Response type
}

// Topic returns the topic name for the PlayerAddedToRoundEvent.
func (e PlayerAddedToRoundEvent) Topic() string {
	return "player_added_to_round"
}

// PlayerRemovedFromRoundEvent is triggered when a player is removed from a round.
type PlayerRemovedFromRoundEvent struct {
	RoundID int64  `json:"round_id"`
	UserID  string `json:"user_id"`
}

// Topic returns the topic name for the PlayerRemovedFromRoundEvent.
func (e PlayerRemovedFromRoundEvent) Topic() string {
	return "player_removed_from_round"
}

// ScoreSubmittedEvent is triggered when a player submits their score for a round.
type ScoreSubmittedEvent struct {
	RoundID int64  `json:"round_id"`
	UserID  string `json:"user_id"`
	Score   int    `json:"score"`
}

// Topic returns the topic name for the ScoreSubmittedEvent.
func (e ScoreSubmittedEvent) Topic() string {
	return "score_submitted"
}

// TagNumberRequestedEvent is triggered when a player is added to a round and their tag number is needed.
type TagNumberRequestedEvent struct {
	UserID  string `json:"user_id"`
	RoundID int64  `json:"round_id"`
}

// Topic returns the topic name for the TagNumberRequestedEvent.
func (e TagNumberRequestedEvent) Topic() string {
	return "tag_number_requested"
}

// TagNumberRetrievedEvent is triggered when a player's tag number is retrieved.
type TagNumberRetrievedEvent struct {
	UserID    string `json:"user_id"`
	RoundID   int64  `json:"round_id"`
	TagNumber int    `json:"tag_number"`
}

// Topic returns the topic name for the TagNumberRetrievedEvent.
func (e TagNumberRetrievedEvent) Topic() string {
	return "tag_number_retrieved"
}

// Marshal marshals the RoundFinalizedEvent into a JSON byte array.
func (e RoundFinalizedEvent) Marshal() []byte {
	data, _ := json.Marshal(e) // Consider handling the error here
	return data
}

// RoundFinalizedEvent is triggered when a round is finalized.
type RoundFinalizedEvent struct {
	RoundID      int64              `json:"round_id"`
	Participants []ParticipantScore `json:"participants"`
}

// Topic returns the topic name for the RoundFinalizedEvent.
func (e RoundFinalizedEvent) Topic() string {
	return "round_finalized"
}

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
