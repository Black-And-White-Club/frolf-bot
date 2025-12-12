package rounddb

import (
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// Round represents a single round in the tournament.
type Round struct {
	bun.BaseModel  `bun:"table:rounds,alias:r"`
	ID             sharedtypes.RoundID      `bun:"id,pk,type:uuid"`
	Title          roundtypes.Title         `bun:"title,notnull"`
	Description    roundtypes.Description   `bun:"description,nullzero"`
	Location       roundtypes.Location      `bun:"location,nullzero"`
	EventType      *roundtypes.EventType    `bun:"event_type,default:'casual'"`
	StartTime      sharedtypes.StartTime    `bun:"start_time,notnull"`
	Finalized      roundtypes.Finalized     `bun:"finalized,notnull"`
	CreatedBy      sharedtypes.DiscordID    `bun:"created_by,notnull"`
	State          roundtypes.RoundState    `bun:"state,notnull"`
	Participants   []roundtypes.Participant `bun:"participants,type:jsonb"`
	EventMessageID string                   `bun:"event_message_id,nullzero"`
	GuildID        sharedtypes.GuildID      `bun:"guild_id,notnull"`
	CreatedAt      time.Time                `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt      time.Time                `bun:",nullzero,notnull,default:current_timestamp"`
	// Import/scorecard fields
	ImportID        string       `bun:"import_id,nullzero"`
	ImportStatus    ImportStatus `bun:"import_status,nullzero"`
	ImportType      ImportType   `bun:"import_type,nullzero"`
	FileData        []byte       `bun:"file_data,type:bytea"`
	FileName        string       `bun:"file_name,nullzero"`
	UDiscURL        string       `bun:"u_disc_url,nullzero"`
	ImportNotes     string       `bun:"import_notes,nullzero"`
	ImportError     string       `bun:"import_error,nullzero"`
	ImportErrorCode string       `bun:"import_error_code,nullzero"`
	ImportedAt      *time.Time   `bun:"imported_at,type:timestamp"`
	// Import context - who initiated and where to respond
	ImportUserID    sharedtypes.DiscordID `bun:"import_user_id,nullzero"`
	ImportChannelID string                `bun:"import_channel_id,nullzero"`
}

// Response represents the possible responses for a participant.
type Response string

// Define the possible response values as constants.
const (
	ResponseAccept    Response = "ACCEPT"
	ResponseTentative Response = "TENTATIVE"
	ResponseDecline   Response = "DECLINE"
)

// RoundState represents the state of a round.
type RoundState string

// Enum constants for RoundState
const (
	RoundStateUpcoming   RoundState = "UPCOMING"
	RoundStateInProgress RoundState = "IN_PROGRESS"
	RoundStateFinalized  RoundState = "FINALIZED"
	RoundStateDeleted    RoundState = "DELETED"
)

// ImportStatus represents the status of a scorecard import.
type ImportStatus string

const (
	ImportStatusPending    ImportStatus = "pending"
	ImportStatusProcessing ImportStatus = "processing"
	ImportStatusParsing    ImportStatus = "parsing"
	ImportStatusMatching   ImportStatus = "matching"
	ImportStatusCompleted  ImportStatus = "completed"
	ImportStatusFailed     ImportStatus = "failed"
)

// ImportType represents the type of scorecard import.
type ImportType string

const (
	ImportTypeCSV  ImportType = "csv"
	ImportTypeXLSX ImportType = "xlsx"
	ImportTypeURL  ImportType = "url"
)

// Participant represents a user participating in a round.
type Participant struct {
	UserID    sharedtypes.DiscordID  `json:"user_id"`
	TagNumber *sharedtypes.TagNumber `json:"tag_number"`
	Response  Response               `json:"response"`
	Score     *sharedtypes.Score     `json:"score"`
}
