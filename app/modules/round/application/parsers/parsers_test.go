package parsers

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestFactory_GetParser(t *testing.T) {
	factory := NewFactory()
	tests := []struct {
		name     string
		filename string
		want     string
		wantErr  bool
	}{
		{name: "csv file", filename: "scores.csv", want: "csv"},
		{name: "xlsx file", filename: "scores.xlsx", want: "xlsx"},
		{name: "xls file", filename: "scores.xls", want: "xlsx"},
		{name: "unsupported file", filename: "scores.txt", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser, err := factory.GetParser(tt.filename)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			switch tt.want {
			case "csv":
				_, ok := parser.(*CSVParser)
				require.True(t, ok)
			case "xlsx":
				_, ok := parser.(*XLSXParser)
				require.True(t, ok)
			default:
				t.Fatalf("unexpected parser type %q", tt.want)
			}
		})
	}
}

func TestCSVParser_Parse(t *testing.T) {
	parser := NewCSVParser()
	tests := []struct {
		name       string
		data       string
		wantErr    bool
		wantPar    []int
		wantPlayer int
	}{
		{
			name:       "labeled par row",
			data:       "Name,1,2,3,4,5,6,7,8,9\nPar,3,4,3,4,3,4,3,4,3\nPlayer One,3,4,3,4,3,4,3,4,3\nPlayer Two,4,4,4,4,4,4,4,4,4",
			wantPar:    []int{3, 4, 3, 4, 3, 4, 3, 4, 3},
			wantPlayer: 2,
		},
		{
			name:       "numeric par row",
			data:       "Name,1,2,3,4,5,6,7,8,9\n3,4,3,4,3,4,3,4,3\nPlayer One,3,4,3,4,3,4,3,4,3",
			wantPar:    []int{3, 4, 3, 4, 3, 4, 3, 4, 3},
			wantPlayer: 1,
		},
		{
			name:    "invalid par row",
			data:    "Name,1,2\nPar,3,not-a-number",
			wantErr: true,
		},
		{
			name:    "no players",
			data:    "Name,1,2,3\nPar,3,3,3",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse([]byte(tt.data))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantPar, result.ParScores)
			require.Len(t, result.PlayerScores, tt.wantPlayer)
		})
	}
}

func TestXLSXParser_Parse(t *testing.T) {
	parser := NewXLSXParser()
	tests := []struct {
		name                string
		rows                [][]string
		wantErr             bool
		wantPlayerCount     int
		wantFirstPlayerName string
	}{
		{
			name: "normal sheet",
			rows: [][]string{
				{"Name", "1", "2", "3"},
				{"Par", "3", "3", "3"},
				{"Player One", "3", "4", "3", "10"},
				{"Player Two", "4", "4", "4", "12"},
			},
			wantPlayerCount: 2,
		},
		{
			name: "numeric par row without label",
			rows: [][]string{
				{"Name", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
				{"3", "3", "3", "3", "3", "3", "3", "3", "3"},
				{"Player One", "3", "4", "3", "4", "3", "4", "3", "4", "3"},
			},
			wantPlayerCount: 1,
		},
		{
			name: "leaderboard username column",
			rows: [][]string{
				{"Division", "Position", "Username", "Hole 1", "Hole 2"},
				{"Open", "1", "CoolDuck", "3", "2"},
				{"Open", "2", "OtherUser", "4", "3"},
			},
			wantPlayerCount:     2,
			wantFirstPlayerName: "CoolDuck",
		},
		{
			name: "missing par row",
			rows: [][]string{
				{"Name", "1", "2", "3"},
				{"Player One", "3", "4", "3", "10"},
			},
			wantErr: true,
		},
		{
			name:    "empty sheet",
			rows:    [][]string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := buildXLSX(t, tt.rows)
			result, err := parser.Parse(data)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, result.ParScores)
			require.Equal(t, tt.wantPlayerCount, len(result.PlayerScores))
			if tt.wantFirstPlayerName != "" {
				require.Equal(t, tt.wantFirstPlayerName, result.PlayerScores[0].PlayerName)
			}
		})
	}
}

func Test_isLikelyPlayerNameXLSX(t *testing.T) {
	t.Run("numeric is not a name", func(t *testing.T) {
		require.False(t, isLikelyPlayerNameXLSX("123"))
		require.False(t, isLikelyPlayerNameXLSX("  7  "))
	})

	t.Run("non-numeric looks like a name", func(t *testing.T) {
		require.True(t, isLikelyPlayerNameXLSX("Player One"))
		require.True(t, isLikelyPlayerNameXLSX("A"))
	})
}

func buildXLSX(t *testing.T, rows [][]string) []byte {
	f := excelize.NewFile()
	sheet := f.GetSheetName(f.GetActiveSheetIndex())
	for idx, row := range rows {
		axis, err := excelize.CoordinatesToCellName(1, idx+1)
		require.NoError(t, err)
		cells := make([]interface{}, len(row))
		for i, val := range row {
			cells[i] = val
		}
		require.NoError(t, f.SetSheetRow(sheet, axis, &cells))
	}
	var buf bytes.Buffer
	require.NoError(t, f.Write(&buf))
	require.NoError(t, f.Close())
	return buf.Bytes()
}
