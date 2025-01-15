package roundutil

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DateTimeParser interface for parsing date/time input
type DateTimeParser interface {
	ParseDateTime(input string) (time.Time, error)
}

// dateTimeParser implements DateTimeParser
type dateTimeParser struct{}

// NewDateTimeParser creates a new DateTimeParser
func NewDateTimeParser() DateTimeParser {
	return &dateTimeParser{}
}

// Precompiled regular expressions for performance
var (
	tomorrowRegex        = regexp.MustCompile(`^tomorrow at (\d{1,2})(?::(\d{2}))?(am|pm)?$`)
	nextWeekdayRegex     = regexp.MustCompile(`^next (\w+) at (\d{1,2})(?::(\d{2}))?(am|pm)?$`)
	specificDateRegex    = regexp.MustCompile(`^\d{1,2}/\d{1,2}/\d{2,4} at \d{1,2}:\d{2}(am|pm)?$`)
	specificDateISORegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} at \d{2}:\d{2}$`)
)

// ParseDateTime parses a date/time string from user input into a time.Time.
func (p *dateTimeParser) ParseDateTime(input string) (time.Time, error) { // Now a method of dateTimeParser
	// 1. Normalize the input
	normalizedInput := p.normalizeDateTimeInput(input)

	// 2. Use a switch statement for parsing
	switch {
	case p.isRelativeDate(normalizedInput):
		return p.parseRelativeDateTimeInput(normalizedInput)
	case p.isSpecificDate(normalizedInput):
		return p.parseSpecificDateTimeInput(normalizedInput)
	}

	// 3. Try parsing with different formats (fail fast)
	if parsedTime, err := p.parseWithFormat(normalizedInput, "01/02/06 at 3:04 PM"); err == nil {
		return parsedTime, nil
	}
	if parsedTime, err := p.parseWithFormat(normalizedInput, "2006-01-02 at 15:04"); err == nil {
		return parsedTime, nil
	}
	if parsedTime, err := p.parseWithFormat(normalizedInput, "2006-01-02 15:04"); err == nil {
		return parsedTime, nil
	}

	return time.Time{}, fmt.Errorf("invalid date/time format: %s", input)
}

// normalizeDateTimeInput preprocesses the input string.
func (p *dateTimeParser) normalizeDateTimeInput(input string) string {
	input = strings.ToLower(input)
	input = strings.TrimSpace(input)
	return input
}

// parseRelativeDateTimeInput parses relative date/time expressions.
func (p *dateTimeParser) parseRelativeDateTimeInput(input string) (time.Time, error) {
	now := time.Now()

	if tomorrowRegex.MatchString(input) {
		matches := tomorrowRegex.FindStringSubmatch(input)
		return p.parseTimeFromMatches(matches, now.AddDate(0, 0, 1))
	} else if nextWeekdayRegex.MatchString(input) { // Use else if
		matches := nextWeekdayRegex.FindStringSubmatch(input)
		weekday, err := p.parseWeekday(matches[1])
		if err != nil {
			return time.Time{}, err
		}

		// Calculate days until the next weekday
		daysUntilWeekday := int(weekday - now.Weekday())
		if daysUntilWeekday <= 0 {
			daysUntilWeekday += 7
		}

		nextWeekday := now.AddDate(0, 0, daysUntilWeekday)
		return p.parseTimeFromMatches(matches, nextWeekday)
	}

	return time.Time{}, fmt.Errorf("invalid relative date/time format")
}

// parseTimeFromMatches extracts hours and minutes from regex matches and returns the corresponding time.
func (p *dateTimeParser) parseTimeFromMatches(matches []string, baseTime time.Time) (time.Time, error) {
	hour, err := p.parseIntWithError(matches[1])
	if err != nil {
		return time.Time{}, err // Return error immediately
	}
	minute := 0
	if matches[2] != "" {
		minute, err = p.parseIntWithError(matches[2])
		if err != nil {
			return time.Time{}, err
		}
	}
	if matches[3] == "pm" && hour < 12 {
		hour += 12
	} else if matches[3] == "am" && hour == 12 {
		hour = 0
	}
	return baseTime.Truncate(24 * time.Hour).Add(time.Hour*time.Duration(hour) + time.Minute*time.Duration(minute)), nil
}

// parseSpecificDateTimeInput parses specific date/time expressions.
func (p *dateTimeParser) parseSpecificDateTimeInput(input string) (time.Time, error) {
	formats := map[*regexp.Regexp]string{
		specificDateRegex:    "01/02/06 at 3:04pm",
		specificDateISORegex: "2006-01-02 at 15:04",
	}
	for regex, format := range formats {
		if regex.MatchString(input) {
			parsedTime, err := time.Parse(format, input)
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid date format: expected '%s'", format)
			}
			return parsedTime, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid specific date/time format")
}

// parseWithFormat parses the input using the given format string.
func (p *dateTimeParser) parseWithFormat(input string, format string) (time.Time, error) {
	parsedTime, err := time.Parse(format, input)
	if err != nil {
		return time.Time{}, err
	}
	return parsedTime, nil
}

// isRelativeDate checks if the input contains relative date keywords.
func (p *dateTimeParser) isRelativeDate(input string) bool {
	return tomorrowRegex.MatchString(input) || nextWeekdayRegex.MatchString(input)
}

// parseWeekday parses a weekday string into a time.Weekday.
func (p *dateTimeParser) parseWeekday(weekday string) (time.Weekday, error) {
	switch weekday {
	case "sunday":
		return time.Sunday, nil
	case "monday":
		return time.Monday, nil
	case "tuesday":
		return time.Tuesday, nil
	case "wednesday":
		return time.Wednesday, nil
	case "thursday":
		return time.Thursday, nil
	case "friday":
		return time.Friday, nil
	case "saturday":
		return time.Saturday, nil
	default:
		return time.Sunday, fmt.Errorf("invalid weekday: %s", weekday)
	}
}

// parseIntWithError parses an integer from a string and returns an error if it fails.
func (p *dateTimeParser) parseIntWithError(str string) (int, error) {
	i, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("failed to parse integer from '%s': %w", str, err)
	}
	return i, nil
}

// isSpecificDate checks if the input matches a specific date pattern.
func (p *dateTimeParser) isSpecificDate(input string) bool {
	return specificDateRegex.MatchString(input) || specificDateISORegex.MatchString(input)
}
