package roundtime

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/olebedev/when"
	"github.com/olebedev/when/rules/en"
)

// TimeParserInterface defines the methods for time parsing and timezone handling.
type TimeParserInterface interface {
	GetTimezoneFromInput(input string) (string, bool)
	ParseUserTimeInput(startTimeStr string, timezone roundtypes.Timezone, clock roundutil.Clock) (int64, error)
}

// TimeParser struct holds the timezone mappings and implements TimeParserInterface.
type TimeParser struct {
	TimezoneMap map[string]string
}

// NewTimeParser creates a new TimeParser instance with predefined timezone mappings.
func NewTimeParser() *TimeParser {
	return &TimeParser{
		TimezoneMap: map[string]string{
			"PST": "America/Los_Angeles",
			"PDT": "America/Los_Angeles",
			"MST": "America/Denver",
			"MDT": "America/Denver",
			"CST": "America/Chicago",
			"CDT": "America/Chicago",
			"EST": "America/New_York",
			"EDT": "America/New_York",
		},
	}
}

// GetTimezoneFromInput extracts a US timezone abbreviation from user input.
func (tp *TimeParser) GetTimezoneFromInput(input string) (string, bool) {
	// Handle empty input first
	if strings.TrimSpace(input) == "" {
		slog.Warn("Empty timezone input, falling back to default", slog.String("input", input))
		return "America/Chicago", false
	}

	// First try as a valid full timezone
	if _, err := time.LoadLocation(input); err == nil {
		return input, true
	}

	// Then try abbreviation map
	inputUpper := strings.ToUpper(input)
	for abbreviation, fullName := range tp.TimezoneMap {
		if strings.Contains(inputUpper, abbreviation) {
			return fullName, true
		}
	}

	// Fallback to default timezone
	slog.Warn("Unknown timezone, falling back to default", slog.String("input", input))
	return "America/Chicago", false
}

// ParseUserTimeInput parses user-provided time and converts it to a UTC timestamp.
func (tp *TimeParser) ParseUserTimeInput(startTimeStr string, timezone roundtypes.Timezone, clock roundutil.Clock) (int64, error) {
	// Validate start time string
	if strings.TrimSpace(startTimeStr) == "" {
		return 0, fmt.Errorf("start time string cannot be empty")
	}

	// Determine the timezone (allows fallback)
	userTimeZone, _ := tp.GetTimezoneFromInput(string(timezone))
	// Note: We don't check the 'found' boolean since we allow fallback
	slog.Info("Timezone override", slog.String("user_timezone", userTimeZone))

	// Load the timezone
	loc, err := time.LoadLocation(userTimeZone)
	if err != nil {
		return 0, fmt.Errorf("failed to load timezone: %s", userTimeZone)
	}

	// Normalize the input
	startTimeStr = strings.ToLower(startTimeStr)
	startTimeStr = strings.ReplaceAll(startTimeStr, "today ", "today at ")

	// Ensure time format includes a colon (e.g., "932am" → "9:32 AM")
	timePattern := `(\d{1,2})(\d{2})(am|pm)`
	startTimeStr = regexp.MustCompile(timePattern).ReplaceAllString(startTimeStr, "$1:$2 $3")

	// Initialize `when` parser
	w := when.New(nil)
	w.Add(en.All...)

	// Try parsing with `when`
	r, err := w.Parse(startTimeStr, clock.Now().In(loc))
	if err != nil {
		slog.Error("Error parsing time input with when", slog.String("input", startTimeStr), slog.Any("error", err))
	}
	if r != nil {
		parsedTime := r.Time.In(loc)
		slog.Info("Parsed time using when", slog.String("parsed_time", parsedTime.Format(time.RFC3339)))

		// Ensure parsed time is in the future
		nowInLoc := clock.Now().In(loc).Truncate(time.Minute)
		parsedTime = parsedTime.Truncate(time.Minute)

		if parsedTime.Before(nowInLoc) {
			return 0, fmt.Errorf("start time must be in the future (parsed: %s, now: %s)", parsedTime, nowInLoc)
		}

		// Convert to UTC and return
		return parsedTime.In(time.UTC).Unix(), nil
	}

	// If `when` fails, try manual parsing
	slog.Warn("`when` failed to parse input, falling back to manual parsing", slog.String("input", startTimeStr))

	// Try parsing as "Monday 9:32 AM"
	manualTimeStr := fmt.Sprintf("%s %s", clock.Now().Weekday().String(), startTimeStr)
	parsedTime, err := time.ParseInLocation("Monday 3:04 PM", manualTimeStr, loc)
	if err != nil {
		return 0, fmt.Errorf("could not recognize time format: %s", startTimeStr)
	}

	slog.Info("Parsed time using manual fallback", slog.String("parsed_time", parsedTime.Format(time.RFC3339)))
	return parsedTime.In(time.UTC).Unix(), nil
}
