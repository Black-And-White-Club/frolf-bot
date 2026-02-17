package roundservice

import "testing"

func TestNormalizeUDiscExportURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "canonical leaderboard URL",
			input: "https://udisc.com/courses/abc/rounds/123/leaderboard",
			want:  "https://udisc.com/courses/abc/rounds/123/leaderboard/export",
		},
		{
			name:  "strips query and fragment",
			input: "https://www.udisc.com/courses/abc/rounds/123/leaderboard?x=1#frag",
			want:  "https://www.udisc.com/courses/abc/rounds/123/leaderboard/export",
		},
		{
			name:    "rejects non-https",
			input:   "http://udisc.com/courses/abc/rounds/123/leaderboard",
			wantErr: true,
		},
		{
			name:    "rejects unsupported host",
			input:   "https://169.254.169.254/latest/meta-data/?x=udisc.com",
			wantErr: true,
		},
		{
			name:    "rejects userinfo",
			input:   "https://user:pass@udisc.com/courses/abc/rounds/123/leaderboard",
			wantErr: true,
		},
		{
			name:    "rejects explicit ports",
			input:   "https://udisc.com:444/courses/abc/rounds/123/leaderboard",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeUDiscExportURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got none (url=%s)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
