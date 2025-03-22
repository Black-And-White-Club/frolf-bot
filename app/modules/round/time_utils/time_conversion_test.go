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
			want:  "",
			want1: false,
		},
		{
			name:  "Mixed case timezone",
			input: "Meeting at 4 PM cst",
			want:  "America/Chicago",
			want1: true,
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
			mockNow:      time.Date(2025, 6, 5, 12, 0, 0, 0, time.UTC),
			want:         time.Date(2025, 6, 7, 0, 0, 0, 0, time.UTC).Unix(),
			wantErr:      false,
		},
		{
			name:         "Invalid timezone",
			startTimeStr: "5 PM",
			timezone:     "XYZ",
			mockNow:      time.Date(2025, 6, 5, 12, 0, 0, 0, time.UTC),
			want:         0,
			wantErr:      true,
		},
		{
			name:         "Invalid date format",
			startTimeStr: "invalid date",
			timezone:     "CST",
			mockNow:      time.Date(2025, 6, 5, 12, 0, 0, 0, time.UTC),
			want:         0,
			wantErr:      true,
		},
		{
			name:         "6am (time already passed)",
			startTimeStr: "6am",
			timezone:     "CDT",
			mockNow:      time.Date(2025, 6, 5, 7, 0, 0, 0, time.UTC), // 2 AM CDT
			want:         0,                                           // Should fail since 6 AM CDT has passed
			wantErr:      true,
		},
		{
			name:         "7pm (time not yet passed)",
			startTimeStr: "7pm",
			timezone:     "CDT",
			mockNow:      time.Date(2025, 6, 5, 12, 0, 0, 0, time.UTC),        // 7 AM CDT
			want:         time.Date(2025, 6, 6, 00, 0, 0, 0, time.UTC).Unix(), // 7 PM CDT -> 00:00 UTC
			wantErr:      false,
		},
		{
			name:         "Empty timezone string",
			startTimeStr: "Tomorrow 5 PM",
			timezone:     "",
			mockNow:      time.Date(2025, 6, 5, 12, 0, 0, 0, time.UTC),
			want:         0,
			wantErr:      true,
		},
		{
			name:         "Malformed time string",
			startTimeStr: "25 PM",
			timezone:     "PDT",
			mockNow:      time.Date(2025, 6, 5, 12, 0, 0, 0, time.UTC),
			want:         0,
			wantErr:      true,
		},
		{
			name:         "DST Transition",
			startTimeStr: "Nov 5 1:30 AM",
			timezone:     "PDT",
			mockNow:      time.Date(2025, 11, 4, 12, 0, 0, 0, time.UTC),
			want:         time.Date(2025, 11, 5, 9, 30, 0, 0, time.UTC).Unix(), // Adjusted for DST
			wantErr:      false,
		},
		{
			name:         "Empty startTimeStr",
			startTimeStr: "",
			timezone:     "PDT",
			mockNow:      time.Date(2025, 6, 5, 12, 0, 0, 0, time.UTC),
			want:         0,
			wantErr:      true,
		},
		{
			name:         "Nil timezone (default handling)",
			startTimeStr: "Tomorrow 5 PM",
			timezone:     "", // Simulates nil behavior for a string
			mockNow:      time.Date(2025, 6, 5, 12, 0, 0, 0, time.UTC),
			want:         0,
			wantErr:      true,
		},
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
				t.Errorf("TimeParser.ParseUserTimeInput() = %v, want %v", got, tt.want)
			}
		})
	}
}
