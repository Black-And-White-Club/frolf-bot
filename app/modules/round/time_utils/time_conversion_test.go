package roundtime

import (
	"reflect"
	"testing"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/round/mocks"
	"go.uber.org/mock/gomock"
)

func TestNewTimeParser(t *testing.T) {
	tests := []struct {
		name string
		want *TimeParser
	}{
		{
			name: "Create TimeParser",
			want: &TimeParser{
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
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTimeParser(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTimeParser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeParser_GetTimezoneFromInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		want1 bool
	}{
		{
			name:  "Valid timezone PST",
			input: "Let's meet at 5 PM PST",
			want:  "America/Los_Angeles",
			want1: true,
		},
		{
			name:  "Valid timezone EST",
			input: "Schedule for 3 PM EST",
			want:  "America/New_York",
			want1: true,
		},
		{
			name:  "Invalid timezone",
			input: "What time is it?",
			want:  "America/Chicago", // Fallback timezone
			want1: false,             // Indicates fallback was used
		},
		{
			name:  "Mixed case timezone",
			input: "Meeting at 4 PM cst",
			want:  "America/Chicago",
			want1: true,
		},
		{
			name:  "Valid full timezone name",
			input: "America/New_York",
			want:  "America/New_York",
			want1: true,
		},
		{
			name:  "Unknown abbreviation",
			input: "XYZ",
			want:  "America/Chicago", // Fallback timezone
			want1: false,             // Indicates fallback was used
		},
	}
	tp := NewTimeParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tp.GetTimezoneFromInput(tt.input)
			if got != tt.want {
				t.Errorf("TimeParser.GetTimezoneFromInput() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("TimeParser.GetTimezoneFromInput() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestTimeParser_ParseUserTimeInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClock := mocks.NewMockClock(ctrl)

	tests := []struct {
		name         string
		startTimeStr string
		timezone     roundtypes.Timezone
		mockNow      time.Time
		want         int64
		wantErr      bool
	}{
		{
			name:         "Valid time in PDT",
			startTimeStr: "Tomorrow 5 PM",
			timezone:     "PDT",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC),
			want:         time.Date(2027, 6, 7, 0, 0, 0, 0, time.UTC).Unix(),
			wantErr:      false,
		},
		{
			name:         "Unknown timezone uses fallback",
			startTimeStr: "Tomorrow 5 PM",
			timezone:     "XYZ",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC),
			want:         time.Date(2027, 6, 6, 22, 0, 0, 0, time.UTC).Unix(), // 5 PM CDT = 22:00 UTC
			wantErr:      false,
		},
		{
			name:         "Invalid date format",
			startTimeStr: "invalid date",
			timezone:     "CST",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC),
			want:         0,
			wantErr:      true,
		},
		{
			name:         "6am (time already passed)",
			startTimeStr: "6am",
			timezone:     "CDT",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC), // 7 AM CDT
			want:         0,                                            // Should fail since 6 AM CDT has passed
			wantErr:      true,
		},
		{
			name:         "7pm (time not yet passed)",
			startTimeStr: "7pm",
			timezone:     "CDT",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC),       // 7 AM CDT
			want:         time.Date(2027, 6, 6, 0, 0, 0, 0, time.UTC).Unix(), // 7 PM CDT -> 00:00 UTC next day
			wantErr:      false,
		},
		{
			name:         "Empty timezone string uses fallback",
			startTimeStr: "Tomorrow 5 PM",
			timezone:     "",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC),
			want:         time.Date(2027, 6, 6, 22, 0, 0, 0, time.UTC).Unix(), // 5 PM CDT = 22:00 UTC
			wantErr:      false,
		},
		{
			name:         "Malformed time string",
			startTimeStr: "25 PM",
			timezone:     "PDT",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC),
			want:         0,
			wantErr:      true,
		},
		{
			name:         "DST Transition",
			startTimeStr: "Nov 5 1:30 AM",
			timezone:     "PDT",
			mockNow:      time.Date(2027, 11, 4, 12, 0, 0, 0, time.UTC),
			want:         time.Date(2027, 11, 5, 8, 30, 0, 0, time.UTC).Unix(), // 1:30 AM PST = 8:30 UTC (PST is UTC-8 in November)
			wantErr:      false,
		},
		{
			name:         "Empty startTimeStr",
			startTimeStr: "",
			timezone:     "PDT",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC),
			want:         0,
			wantErr:      true,
		},
		{
			name:         "Whitespace only startTimeStr",
			startTimeStr: "   ",
			timezone:     "PDT",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC),
			want:         0,
			wantErr:      true,
		},
		{
			name:         "Valid time format 932am",
			startTimeStr: "932am",
			timezone:     "CDT",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC),         // 7 AM CDT
			want:         time.Date(2027, 6, 5, 14, 32, 0, 0, time.UTC).Unix(), // 9:32 AM CDT same day
			wantErr:      false,
		},
		{
			name:         "Today at format",
			startTimeStr: "today 3pm",
			timezone:     "EST",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC),        // 8 AM EST
			want:         time.Date(2027, 6, 5, 20, 0, 0, 0, time.UTC).Unix(), // 3 PM EDT = 20:00 UTC (summer time)
			wantErr:      false,
		},
		{
			name:         "ISO-8601 format YYYY-MM-DD HH:MM",
			startTimeStr: "2027-06-20 18:00",
			timezone:     "America/Chicago",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC), // June 5, 2027 12:00 UTC (7 AM CDT)
			want:         time.Date(2027, 6, 20, 23, 0, 0, 0, time.UTC).Unix(), // June 20, 2027 6 PM CDT = June 20 23:00 UTC
			wantErr:      false,
		},
		{
			name:         "ISO-8601 format YYYY-MM-DD HH:MM:SS",
			startTimeStr: "2027-07-15 14:30:00",
			timezone:     "America/New_York",
			mockNow:      time.Date(2027, 7, 10, 12, 0, 0, 0, time.UTC),       // July 10, 2027
			want:         time.Date(2027, 7, 15, 18, 30, 0, 0, time.UTC).Unix(), // July 15, 2027 2:30 PM EDT = 18:30 UTC
			wantErr:      false,
		},
		{
			name:         "MM/DD/YYYY HH:MM format",
			startTimeStr: "08/25/2027 09:15",
			timezone:     "America/Los_Angeles",
			mockNow:      time.Date(2027, 8, 20, 12, 0, 0, 0, time.UTC),       // Aug 20, 2027
			want:         time.Date(2027, 8, 25, 16, 15, 0, 0, time.UTC).Unix(), // Aug 25, 2027 9:15 AM PDT = 16:15 UTC (DST)
			wantErr:      false,
		},
		{
			name:         "ISO-8601 format in the past",
			startTimeStr: "2027-06-01 10:00",
			timezone:     "America/Chicago",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC), // June 5, 2027 (after the requested time)
			want:         0,
			wantErr:      true,
		},
		{
			name:         "Today at 1030pm (no space)",
			startTimeStr: "Today at 1030pm",
			timezone:     "America/Chicago",
			mockNow:      time.Date(2027, 6, 5, 19, 0, 0, 0, time.UTC), // June 5, 2027 7:00 PM UTC = 2:00 PM CST
			want:         time.Date(2027, 6, 6, 3, 30, 0, 0, time.UTC).Unix(), // June 6, 2027 3:30 AM UTC = June 5, 10:30 PM CST
			wantErr:      false,
		},
		{
			name:         "Today at 1030 pm (with space)",
			startTimeStr: "Today at 1030 pm",
			timezone:     "America/Chicago",
			mockNow:      time.Date(2027, 6, 5, 19, 0, 0, 0, time.UTC), // June 5, 2027 7:00 PM UTC = 2:00 PM CST
			want:         time.Date(2027, 6, 6, 3, 30, 0, 0, time.UTC).Unix(), // June 6, 2027 3:30 AM UTC = June 5, 10:30 PM CST
			wantErr:      false,
		},
		// Test YYYY-MM-DD H:MM AM/PM format
		{
			name:         "ISO-8601 with AM/PM - morning",
			startTimeStr: "2027-06-20 9:00 AM",
			timezone:     "America/Chicago",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC),
			want:         time.Date(2027, 6, 20, 14, 0, 0, 0, time.UTC).Unix(), // 9 AM CDT = 14:00 UTC
			wantErr:      false,
		},
		{
			name:         "ISO-8601 with AM/PM - evening",
			startTimeStr: "2027-06-20 6:00 PM",
			timezone:     "America/Los_Angeles",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC),
			want:         time.Date(2027, 6, 21, 1, 0, 0, 0, time.UTC).Unix(), // 6 PM PDT = 01:00 UTC next day
			wantErr:      false,
		},
		// Test MM/DD/YYYY H:MM AM/PM format
		{
			name:         "US date format with AM/PM",
			startTimeStr: "08/25/2027 9:15 PM",
			timezone:     "America/New_York",
			mockNow:      time.Date(2027, 8, 20, 12, 0, 0, 0, time.UTC),
			want:         time.Date(2027, 8, 26, 1, 15, 0, 0, time.UTC).Unix(), // 9:15 PM EDT = 01:15 UTC next day
			wantErr:      false,
		},
		// Test simple time formats (no date)
		{
			name:         "Simple time 3pm",
			startTimeStr: "3pm",
			timezone:     "CST",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC), // 7 AM CDT
			want:         time.Date(2027, 6, 5, 20, 0, 0, 0, time.UTC).Unix(), // 3 PM CDT same day
			wantErr:      false,
		},
		{
			name:         "Simple time 10am",
			startTimeStr: "10am",
			timezone:     "PDT",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC), // 5 AM PDT
			want:         time.Date(2027, 6, 5, 17, 0, 0, 0, time.UTC).Unix(), // 10 AM PDT same day
			wantErr:      false,
		},
		{
			name:         "Time with colon 5:30pm",
			startTimeStr: "5:30pm",
			timezone:     "America/New_York",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC), // 8 AM EDT
			want:         time.Date(2027, 6, 5, 21, 30, 0, 0, time.UTC).Unix(), // 5:30 PM EDT same day
			wantErr:      false,
		},
		// Test natural language variations
		{
			name:         "Next Tuesday",
			startTimeStr: "next Tuesday 3pm",
			timezone:     "CST",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC), // Saturday, June 5
			want:         time.Date(2027, 6, 8, 20, 0, 0, 0, time.UTC).Unix(), // Tuesday June 8, 3 PM CDT
			wantErr:      false,
		},
		{
			name:         "Tomorrow at noon",
			startTimeStr: "tomorrow at noon",
			timezone:     "PDT",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC),
			want:         time.Date(2027, 6, 6, 19, 0, 0, 0, time.UTC).Unix(), // June 6 noon PDT = 19:00 UTC
			wantErr:      false,
		},
		// Test variations with "at"
		{
			name:         "Tomorrow at 3pm",
			startTimeStr: "tomorrow at 3pm",
			timezone:     "CDT",
			mockNow:      time.Date(2027, 6, 5, 12, 0, 0, 0, time.UTC),
			want:         time.Date(2027, 6, 6, 20, 0, 0, 0, time.UTC).Unix(),
			wantErr:      false,
		},
		// Test compact time without space before am/pm (already tested with 932am and 1030pm)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, err := time.LoadLocation("America/Chicago") // Adjust to test timezone
			if err != nil {
				t.Fatal(err)
			}

			mockNowInLoc := tt.mockNow.In(loc)
			mockClock.EXPECT().Now().Return(mockNowInLoc).AnyTimes()

			tp := NewTimeParser()

			got, err := tp.ParseUserTimeInput(tt.startTimeStr, tt.timezone, mockClock)
			if (err != nil) != tt.wantErr {
				t.Errorf("TimeParser.ParseUserTimeInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				gotTime := time.Unix(got, 0).UTC().Format(time.RFC3339)
				wantTime := time.Unix(tt.want, 0).UTC().Format(time.RFC3339)
				t.Errorf("TimeParser.ParseUserTimeInput() = %v (%s), want %v (%s)", got, gotTime, tt.want, wantTime)
			}
		})
	}
}
