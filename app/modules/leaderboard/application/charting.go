package leaderboardservice

import (
	"bytes"
	"time"

	"github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

// GenerateTagHistoryChart produces a PNG line chart of a member's tag history.
func GenerateTagHistoryChart(history []TagHistoryView, palette ChartPalette) ([]byte, error) {
	if len(history) == 0 {
		return renderNoDataPlaceholder(palette)
	}

	// 1. Prepare data series
	// Using time series for x-axis (CreatedAt) and value series for y-axis (TagNumber)
	xValues := make([]time.Time, len(history))
	yValues := make([]float64, len(history))

	for i, entry := range history {
		xValues[i] = entry.CreatedAt
		yValues[i] = float64(entry.TagNumber)
	}

	// 2. Setup Chart
	// Note: We want lower tag numbers to be "higher" on the chart visually,
	// but standard charts put higher values at the top.
	// We can invert the Y-axis range or just let it be standard (lower is lower).
	// For tag ranks, usually #1 is at the top. Let's try to invert via range if possible,
	// or standard behavior: #1 is bottom, #100 is top.
	// Actually, standard is fine for now; users understand "up" is "more", so "down" to #1 is good?
	// Or we can flip it. Let's stick to standard for simplicity unless verified otherwise.

	mainSeries := chart.TimeSeries{
		Name:    "Tag History",
		XValues: xValues,
		YValues: yValues,
		Style: chart.Style{
			StrokeColor: drawing.Color(palette.PrimaryLine), // Obsidian Forest Primary
			StrokeWidth: 2,
			DotWidth:    4,
			DotColor:    drawing.Color(palette.AccentLine), // Gold Accent
		},
	}

	graph := chart.Chart{
		Width:  800,
		Height: 400,
		Background: chart.Style{
			FillColor: drawing.Color(palette.Background),
		},
		Canvas: chart.Style{
			FillColor: drawing.Color(palette.Background),
		},
		XAxis: chart.XAxis{
			Name:           "Date",
			ValueFormatter: chart.TimeValueFormatterWithFormat("2006-01-02"),
			Style: chart.Style{
				FontColor: drawing.Color(palette.TextColor),
			},
		},
		YAxis: chart.YAxis{
			Name: "Tag Number",
			Style: chart.Style{
				FontColor: drawing.Color(palette.TextColor),
			},
			Range: &chart.ContinuousRange{
				Descending: true, // Invert so #1 is at the top
			},
		},
		Series: []chart.Series{mainSeries},
	}

	// 3. Render
	buffer := bytes.NewBuffer([]byte{})
	err := graph.Render(chart.PNG, buffer)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func renderNoDataPlaceholder(palette ChartPalette) ([]byte, error) {
	const (
		width  = 400
		height = 200
		msg    = "No tag history found"
	)

	graph := chart.Chart{
		Width:  width,
		Height: height,
		Background: chart.Style{
			FillColor: drawing.Color(palette.Background),
		},
		Canvas: chart.Style{
			FillColor: drawing.Color(palette.Background),
		},
		Elements: []chart.Renderable{
			func(r chart.Renderer, cb chart.Box, chartDefaults chart.Style) {
				r.SetFontColor(drawing.Color(palette.TextColor))
				r.SetFontSize(12.0)
				tb := r.MeasureText(msg)
				x := (cb.Width() - tb.Width()) / 2
				y := (cb.Height() + tb.Height()) / 2
				r.Text(msg, x, y)
			},
		},
	}
	buffer := bytes.NewBuffer([]byte{})
	err := graph.Render(chart.PNG, buffer)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
