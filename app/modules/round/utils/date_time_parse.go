package roundutil

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Precompiled regular expressions for performance
var (
	tomorrowRegex        = regexp.MustCompile(`^tomorrow at (\d{1,2})(?::(\d{2}))?(am|pm)?$`)
	nextWeekdayRegex     = regexp.MustCompile(`^next (\w+) at (\d{1,2})(?::(\d{2}))?(am|pm)?$`)
	specificDateRegex    = regexp.MustCompile(`^\d{1,2}/\d{1,2}/\d{2,4} at \d{1,2}:\d{2}(am|pm)?$`)
	specificDateISORegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} at \d{2}:\d{2}$`)
)

// parseDateTime parses a date/time string from user input into a time.Time.
func ParseDateTime(input string) (time.Time, error) {
	// 1. Normalize the input
	normalizedInput := normalizeInput(input)

	// 2. Use a switch statement for parsing
	switch {
	case isRelativeDate(normalizedInput):
		return parseRelativeDateTime(normalizedInput)
	case isSpecificDate(normalizedInput):
		return parseSpecificDateTime(normalizedInput)
	}

	// 3. Try parsing with different formats (fail fast)
	if parsedTime, err := parseWithFormat(normalizedInput, "01/02/06 at 3:04 PM"); err == nil {
		return parsedTime, nil
	}
	if parsedTime, err := parseWithFormat(normalizedInput, "2006-01-02 at 15:04"); err == nil {
		return parsedTime, nil
	}

	return time.Time{}, fmt.Errorf("invalid date/time format: %s", input)
}

// normalizeInput preprocesses the input string.
func normalizeInput(input string) string {
	input = strings.ToLower(input)
	input = strings.TrimSpace(input)
	return input
}

// parseRelativeDateTime parses relative date/time expressions.
func parseRelativeDateTime(input string) (time.Time, error) {
	now := time.Now()

	if tomorrowRegex.MatchString(input) {
		matches := tomorrowRegex.FindStringSubmatch(input)
		return parseTimeFromMatches(matches, now.AddDate(0, 0, 1))
	}

	if nextWeekdayRegex.MatchString(input) {
		matches := nextWeekdayRegex.FindStringSubmatch(input)
		weekday, err := parseWeekday(matches[1])
		if err != nil {
			return time.Time{}, err
		}

		// Calculate days until the next weekday
		daysUntilWeekday := int(weekday - now.Weekday())
		if daysUntilWeekday <= 0 {
			daysUntilWeekday += 7
		}

		nextWeekday := now.AddDate(0, 0, daysUntilWeekday)
		return parseTimeFromMatches(matches, nextWeekday)
	}

	return time.Time{}, fmt.Errorf("invalid relative date/time format")
}

// parseTimeFromMatches extracts hours and minutes from regex matches and returns the corresponding time.
func parseTimeFromMatches(matches []string, baseTime time.Time) (time.Time, error) {
	hour, err := parseIntWithError(matches[1])
	if err != nil {
		return time.Time{}, err
	}
	minute := 0
	if matches[2] != "" {
		minute, err = parseIntWithError(matches[2])
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

// parseSpecificDateTime parses specific date/time expressions.
func parseSpecificDateTime(input string) (time.Time, error) {
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
func parseWithFormat(input string, format string) (time.Time, error) {
	parsedTime, err := time.Parse(format, input)
	if err != nil {
		return time.Time{}, err
	}
	return parsedTime, nil
}

// isRelativeDate checks if the input contains relative date keywords.
func isRelativeDate(input string) bool {
	return tomorrowRegex.MatchString(input) || nextWeekdayRegex.MatchString(input)
}

// parseWeekday parses a weekday string into a time.Weekday.
func parseWeekday(weekday string) (time.Weekday, error) {
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
func parseIntWithError(str string) (int, error) {
	i, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("failed to parse integer from '%s': %w", str, err)
	}
	return i, nil
}

// isSpecificDate checks if the input matches a specific date pattern.
func isSpecificDate(input string) bool {
	return specificDateRegex.MatchString(input) || specificDateISORegex.MatchString(input)
}
