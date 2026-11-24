package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/Tencent/WeKnora/internal/utils"
	"github.com/google/uuid"
)

// The chat model can already draw an inline SVG chart directly in its prose.
// That copy is throwaway: it cannot be reopened, referenced, or kept after the
// next turn. save_chart produces the persisted counterpart — one self-contained
// HTML file per chart, registered on the assistant message through the same
// artifact path the sandbox uses, so it shows up as a card and can be reopened
// from the side panel.

// Chart rendering limits. They are deliberately low: a chart that needs more
// than this is unreadable in the panel, and an unbounded input would let one
// tool call build a multi-megabyte SVG.
const (
	maxChartCategories = 60
	maxChartSeries     = 6
	chartCanvasWidth   = 820
	chartCanvasHeight  = 440
)

// chartPalette is a colorblind-tolerant qualitative palette that reads on both
// light and dark backgrounds. Text and rules use currentColor instead, so the
// preview follows the surrounding theme.
var chartPalette = []string{
	"#3F6FE0", "#3FB27F", "#E0A03F", "#D96A6A", "#8A63D2", "#3FA9C9",
}

// SaveChartSeries is one plotted series. Values are aligned with Categories by
// index; a short Values slice leaves the tail categories unfilled.
type SaveChartSeries struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
}

// SaveChartInput is the model-facing schema. It carries the already-computed
// numbers, not a query: the agent runs data_analysis first and passes the rows
// it got back, so the chart can never disagree with the table beside it.
type SaveChartInput struct {
	Title      string            `json:"title" jsonschema:"Chart title shown above the plot. Keep it short; it also names the artifact"`
	ChartType  string            `json:"chart_type" jsonschema:"One of: bar, line, pie. Use bar for comparisons, line for trends over time, pie for composition (single series only)"`
	Categories []string          `json:"categories" jsonschema:"X-axis labels or slice names, in display order"`
	Series     []SaveChartSeries `json:"series" jsonschema:"One entry per series. For pie charts pass exactly one series"`
	XLabel     string            `json:"x_label,omitempty" jsonschema:"Optional x-axis label"`
	YLabel     string            `json:"y_label,omitempty" jsonschema:"Optional y-axis label"`
	Note       string            `json:"note,omitempty" jsonschema:"Optional one-line footnote, e.g. the aggregation口径 or the SQL/filters behind the numbers"`
}

// SaveChartTool renders the input into a self-contained HTML file and registers
// it as a message artifact.
type SaveChartTool struct {
	BaseTool
	fileService interfaces.FileService
	sessionID   string
}

var saveChartToolDefinition = BaseTool{
	name: ToolSaveChart,
	description: "Persist a chart so the user can reopen it later. Pass the numbers you already computed " +
		"(from data_analysis), never a query. The chart is saved as a self-contained HTML artifact and " +
		"appears as a card in the conversation. Use it whenever the user asked for a chart, or when a " +
		"figure is worth keeping; keep a plain table in the answer for one-off answers. " +
		"After calling it, reference the returned file name in your final answer exactly as " +
		"![<title>](<file_name>) so the card is placed where you want it.",
	schema: utils.GenerateSchema[SaveChartInput](),
}

// NewSaveChartTool wires the tool to object storage for this turn.
func NewSaveChartTool(fileService interfaces.FileService, sessionID string) *SaveChartTool {
	return &SaveChartTool{
		BaseTool:    saveChartToolDefinition,
		fileService: fileService,
		sessionID:   sessionID,
	}
}

// Execute renders and persists the chart. A failure here must not fail the
// turn: the caller still has the numbers and can answer with a table, so every
// error path returns a model-readable message.
func (t *SaveChartTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	var input SaveChartInput
	if err := json.Unmarshal(args, &input); err != nil {
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Invalid arguments: %v", err),
		}, nil
	}
	if err := validateSaveChartInput(&input); err != nil {
		return &types.ToolResult{Success: false, Error: err.Error()}, nil
	}
	if t.fileService == nil {
		return &types.ToolResult{
			Success: false,
			Error:   "Chart storage is unavailable in this deployment; answer with a table instead.",
		}, nil
	}

	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok {
		return &types.ToolResult{Success: false, Error: "Missing tenant context"}, nil
	}

	doc := renderChartHTML(&input)
	data := []byte(doc)
	fileName := chartFileName(input.Title)

	storageName := "artifact_" + uuid.NewString() + "_" + fileName
	storagePath, err := t.fileService.SaveBytes(ctx, data, tenantID, storageName, false)
	if err != nil {
		logger.Warnf(ctx, "[Tool][SaveChart] upload failed: session=%s err=%v", t.sessionID, err)
		return &types.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to store the chart: %v", err),
		}, nil
	}

	sum := sha256.Sum256(data)
	now := time.Now().UTC()
	artifact := types.MessageArtifact{
		URL:         storagePath,
		FileName:    fileName,
		FileType:    ".html",
		FileSize:    int64(len(data)),
		ContentHash: hex.EncodeToString(sum[:]),
		SourcePath:  "charts/" + fileName,
		ModTime:     now,
		CreatedAt:   now,
	}

	return &types.ToolResult{
		Success: true,
		Output: fmt.Sprintf(
			"Chart saved as %s (%s, %d series x %d points). "+
				"Reference it in your final answer exactly as ![%s](%s) so the card renders.",
			fileName, input.ChartType, len(input.Series), len(input.Categories),
			input.Title, fileName,
		),
		Data: map[string]interface{}{
			"display_type": "chart",
			"chart_type":   input.ChartType,
			"title":        input.Title,
			"file_name":    fileName,
			"categories":   len(input.Categories),
			"series":       len(input.Series),
		},
		Artifacts: types.MessageArtifacts{artifact},
	}, nil
}

// validateSaveChartInput keeps every rejection actionable for the model.
func validateSaveChartInput(in *SaveChartInput) error {
	in.ChartType = strings.ToLower(strings.TrimSpace(in.ChartType))
	switch in.ChartType {
	case "bar", "line", "pie":
	case "":
		in.ChartType = "bar"
	default:
		return fmt.Errorf("Unsupported chart_type %q: use bar, line or pie", in.ChartType)
	}

	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		in.Title = "Chart"
	}
	if utf8.RuneCountInString(in.Title) > 80 {
		return fmt.Errorf("Title is too long (%d characters): keep it under 80", utf8.RuneCountInString(in.Title))
	}

	if len(in.Categories) == 0 {
		return fmt.Errorf("categories is empty: pass the x-axis labels in display order")
	}
	if len(in.Categories) > maxChartCategories {
		return fmt.Errorf("Too many categories (%d): aggregate to at most %d before charting",
			len(in.Categories), maxChartCategories)
	}
	if len(in.Series) == 0 {
		return fmt.Errorf(`series is empty: pass at least one {"name": ..., "values": [...]} entry`)
	}
	if len(in.Series) > maxChartSeries {
		return fmt.Errorf("Too many series (%d): chart at most %d", len(in.Series), maxChartSeries)
	}
	if in.ChartType == "pie" && len(in.Series) != 1 {
		return fmt.Errorf("pie charts take exactly one series (got %d)", len(in.Series))
	}

	for i := range in.Series {
		series := &in.Series[i]
		series.Name = strings.TrimSpace(series.Name)
		if series.Name == "" {
			series.Name = fmt.Sprintf("Series %d", i+1)
		}
		if len(series.Values) == 0 {
			return fmt.Errorf("series %q has no values", series.Name)
		}
		if len(series.Values) > len(in.Categories) {
			return fmt.Errorf("series %q has %d values but only %d categories",
				series.Name, len(series.Values), len(in.Categories))
		}
		for _, v := range series.Values {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return fmt.Errorf("series %q contains a non-finite value", series.Name)
			}
		}
	}
	return nil
}

// chartFileName builds a human-readable, storage-safe name. Non-ASCII titles
// are kept (the artifact card and the reference rewriter both handle them);
// only path separators and characters that break Markdown links are replaced.
func chartFileName(title string) string {
	var b strings.Builder
	for _, r := range title {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 0x4E00 && r <= 0x9FFF, r >= 0x3040 && r <= 0x30FF, r >= 0xAC00 && r <= 0xD7AF:
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	slug := strings.Trim(b.String(), "_")
	if slug == "" {
		slug = "chart"
	}
	runes := []rune(slug)
	if len(runes) > 40 {
		slug = string(runes[:40])
	}
	return slug + "_" + uuid.NewString()[:6] + ".html"
}

// renderChartHTML builds a self-contained document: inline SVG, inline CSS, no
// script and no network dependency, so it renders identically offline and in a
// sandboxed iframe.
func renderChartHTML(in *SaveChartInput) string {
	var body string
	switch in.ChartType {
	case "pie":
		body = renderPieChart(in)
	case "line":
		body = renderLineChart(in)
	default:
		body = renderBarChart(in)
	}

	var caption strings.Builder
	if note := strings.TrimSpace(in.Note); note != "" {
		caption.WriteString(`<p class="note">`)
		caption.WriteString(html.EscapeString(note))
		caption.WriteString(`</p>`)
	}

	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + html.EscapeString(in.Title) + `</title>
<style>
:root {
  color-scheme: light dark;
  --chart-surface: #ffffff;
  --chart-ink: #1f2329;
}
@media (prefers-color-scheme: dark) {
  :root { --chart-surface: #1b1d21; --chart-ink: #e6e8eb; }
}
* { box-sizing: border-box; }
body {
  margin: 0; padding: 24px 20px 20px;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC",
    "Hiragino Sans GB", "Microsoft YaHei", Roboto, sans-serif;
  background: var(--chart-surface); color: var(--chart-ink);
}
h1 { font-size: 17px; font-weight: 600; margin: 0 0 12px; }
svg { display: block; width: 100%; height: auto; }
.note { margin: 10px 2px 0; font-size: 12px; line-height: 1.6; opacity: .62; }
</style>
</head>
<body>
<h1>` + html.EscapeString(in.Title) + `</h1>
` + body + `
` + caption.String() + `
</body>
</html>`
}

// chartFrame holds the plot geometry shared by the cartesian renderers.
type chartFrame struct {
	width, height    float64
	left, right, top float64
	bottom           float64
	plotW, plotH     float64
}

func newChartFrame(in *SaveChartInput) chartFrame {
	f := chartFrame{
		width:  chartCanvasWidth,
		height: chartCanvasHeight,
		left:   76,
		right:  28,
		top:    56,
		bottom: 68,
	}
	f.plotW = f.width - f.left - f.right
	f.plotH = f.height - f.top - f.bottom
	if len(in.Series) > 1 {
		// Reserve room for the legend row under the title.
		f.top += 18
		f.plotH = f.height - f.top - f.bottom
	}
	return f
}

func renderBarChart(in *SaveChartInput) string {
	f := newChartFrame(in)
	minV, maxV := 0.0, 0.0
	for _, s := range in.Series {
		for _, v := range s.Values {
			minV = math.Min(minV, v)
			maxV = math.Max(maxV, v)
		}
	}
	if minV == 0 && maxV == 0 {
		maxV = 1
	}
	lo, hi, step := niceScale(minV, maxV)
	y := func(v float64) float64 { return f.top + f.plotH*(hi-v)/(hi-lo) }

	var b strings.Builder
	b.WriteString(svgOpen(f))
	b.WriteString(renderLegend(in, f))
	b.WriteString(renderYAxis(f, lo, hi, step, y, in.YLabel))

	groupW := f.plotW / float64(len(in.Categories))
	barW := groupW * 0.72 / float64(len(in.Series))

	for gi, category := range in.Categories {
		groupX := f.left + groupW*float64(gi)
		for si, s := range in.Series {
			if gi >= len(s.Values) {
				continue
			}
			v := s.Values[gi]
			x := groupX + groupW*0.14 + barW*float64(si)
			top := y(math.Max(v, 0))
			height := math.Abs(y(v) - y(0))
			if height < 0.5 {
				height = 0.5
			}
			b.WriteString(fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="2" fill="%s"/>`,
				x, top, barW-1.5, height, chartColor(si)))
			if len(in.Series) == 1 && len(in.Categories) <= 14 {
				b.WriteString(fmt.Sprintf(
					`<text x="%.2f" y="%.2f" text-anchor="middle" font-size="11" fill="currentColor" opacity=".72">%s</text>`,
					x+(barW-1.5)/2, top-6, escapeText(fmtNum(v))))
			}
		}
		b.WriteString(renderXLabel(f, groupX+groupW/2, category, len(in.Categories)))
	}

	b.WriteString(renderXAxisTitle(f, in.XLabel))
	b.WriteString(`</svg>`)
	return b.String()
}

func renderLineChart(in *SaveChartInput) string {
	f := newChartFrame(in)
	minV, maxV := math.Inf(1), math.Inf(-1)
	for _, s := range in.Series {
		for _, v := range s.Values {
			minV = math.Min(minV, v)
			maxV = math.Max(maxV, v)
		}
	}
	if math.IsInf(minV, 1) || minV == maxV {
		minV, maxV = minV-1, maxV+1
	}
	// A little headroom keeps markers off the frame.
	pad := (maxV - minV) * 0.08
	lo, hi, step := niceScale(minV-pad, maxV+pad)
	y := func(v float64) float64 { return f.top + f.plotH*(hi-v)/(hi-lo) }
	x := func(i int) float64 {
		if len(in.Categories) == 1 {
			return f.left + f.plotW/2
		}
		return f.left + f.plotW*float64(i)/float64(len(in.Categories)-1)
	}

	var b strings.Builder
	b.WriteString(svgOpen(f))
	b.WriteString(renderLegend(in, f))
	b.WriteString(renderYAxis(f, lo, hi, step, y, in.YLabel))

	for si, s := range in.Series {
		color := chartColor(si)
		var points []string
		for i, v := range s.Values {
			points = append(points, fmt.Sprintf("%.2f,%.2f", x(i), y(v)))
		}
		b.WriteString(fmt.Sprintf(
			`<polyline fill="none" stroke="%s" stroke-width="2.2" stroke-linejoin="round" stroke-linecap="round" points="%s"/>`,
			color, strings.Join(points, " ")))
		if len(s.Values) <= 40 {
			for i, v := range s.Values {
				b.WriteString(fmt.Sprintf(`<circle cx="%.2f" cy="%.2f" r="3" fill="%s"/>`, x(i), y(v), color))
			}
		}
	}

	step2 := 1
	if len(in.Categories) > 12 {
		step2 = int(math.Ceil(float64(len(in.Categories)) / 12))
	}
	for i, category := range in.Categories {
		if i%step2 != 0 {
			continue
		}
		b.WriteString(renderXLabel(f, x(i), category, len(in.Categories)))
	}

	b.WriteString(renderXAxisTitle(f, in.XLabel))
	b.WriteString(`</svg>`)
	return b.String()
}

func renderPieChart(in *SaveChartInput) string {
	f := newChartFrame(in)
	values := in.Series[0].Values
	total := 0.0
	for _, v := range values {
		total += math.Abs(v)
	}
	if total == 0 {
		total = 1
	}

	cx := f.left + f.plotW/2
	cy := f.top + f.plotH/2
	radius := math.Min(f.plotW, f.plotH)*0.38 - 6
	if radius < 30 {
		radius = 30
	}

	var b strings.Builder
	b.WriteString(svgOpen(f))

	angle := -math.Pi / 2
	for i, v := range values {
		sweep := math.Abs(v) / total * 2 * math.Pi
		if sweep == 0 {
			continue
		}
		if sweep >= 2*math.Pi-1e-9 {
			// A single non-zero value: an arc whose start equals its end draws
			// nothing, so a full disc is the only correct shape.
			b.WriteString(fmt.Sprintf(`<circle cx="%.2f" cy="%.2f" r="%.2f" fill="%s"/>`,
				cx, cy, radius, chartColor(i)))
			break
		}
		end := angle + sweep
		path := fmt.Sprintf(`<path d="%s" fill="%s"/>`, pieSlicePath(cx, cy, radius, angle, end), chartColor(i))
		b.WriteString(path)
		// Label slices that are wide enough to hold text.
		if sweep > 0.28 {
			mid := angle + sweep/2
			lx := cx + math.Cos(mid)*radius*0.62
			ly := cy + math.Sin(mid)*radius*0.62
			b.WriteString(fmt.Sprintf(
				`<text x="%.2f" y="%.2f" text-anchor="middle" dominant-baseline="middle" font-size="12" fill="#ffffff">%s</text>`,
				lx, ly, escapeText(fmtPct(sweep/(2*math.Pi)))))
		}
		angle = end
	}
	// Donut hole, so the centre reads as a composition rather than a solid disc.
	b.WriteString(fmt.Sprintf(`<circle cx="%.2f" cy="%.2f" r="%.2f" fill="var(--chart-surface)"/>`,
		cx, cy, radius*0.42))
	b.WriteString(fmt.Sprintf(
		`<text x="%.2f" y="%.2f" text-anchor="middle" dominant-baseline="middle" font-size="13" fill="var(--chart-ink)">%s</text>`,
		cx, cy, escapeText(fmtNum(total))))

	// Legend to the right of the donut: slice names with their share.
	legendX := f.left + f.plotW*0.78
	legendY := f.top + 24
	for i, category := range in.Categories {
		if i >= len(values) || i >= len(chartPalette)+2 {
			break
		}
		share := 0.0
		if total != 0 {
			share = math.Abs(values[i]) / total
		}
		b.WriteString(fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="9" height="9" rx="2" fill="%s"/>`,
			legendX, legendY-8, chartColor(i)))
		b.WriteString(fmt.Sprintf(
			`<text x="%.2f" y="%.2f" font-size="12" fill="currentColor">%s · %s</text>`,
			legendX+15, legendY, escapeText(truncateText(category, 14)), escapeText(fmtPct(share))))
		legendY += 22
	}

	b.WriteString(`</svg>`)
	return b.String()
}

func pieSlicePath(cx, cy, r, from, to float64) string {
	x1 := cx + math.Cos(from)*r
	y1 := cy + math.Sin(from)*r
	x2 := cx + math.Cos(to)*r
	y2 := cy + math.Sin(to)*r
	large := 0
	if to-from > math.Pi {
		large = 1
	}
	return fmt.Sprintf("M%.2f,%.2f L%.2f,%.2f A%.2f,%.2f 0 %d 1 %.2f,%.2f Z",
		cx, cy, x1, y1, r, r, large, x2, y2)
}

func svgOpen(f chartFrame) string {
	return fmt.Sprintf(
		`<svg viewBox="0 0 %.0f %.0f" role="img" xmlns="http://www.w3.org/2000/svg" `+
			`font-family="-apple-system,BlinkMacSystemFont,Segoe UI,PingFang SC,Microsoft YaHei,sans-serif">`,
		f.width, f.height)
}

func renderLegend(in *SaveChartInput, f chartFrame) string {
	if len(in.Series) < 2 {
		return ""
	}
	var b strings.Builder
	x := f.left
	y := f.top - 24
	for i, s := range in.Series {
		if i > 4 {
			break
		}
		b.WriteString(fmt.Sprintf(`<rect x="%.2f" y="%.2f" width="10" height="10" rx="2" fill="%s"/>`,
			x, y-8, chartColor(i)))
		b.WriteString(fmt.Sprintf(`<text x="%.2f" y="%.2f" font-size="12" fill="currentColor" opacity=".8">%s</text>`,
			x+15, y, escapeText(truncateText(s.Name, 16))))
		x += 26 + float64(utf8.RuneCountInString(truncateText(s.Name, 16)))*12
	}
	return b.String()
}

func renderYAxis(f chartFrame, lo, hi, step float64, y func(float64) float64, label string) string {
	var b strings.Builder
	for v := lo; v <= hi+step/2; v += step {
		yy := y(v)
		b.WriteString(fmt.Sprintf(
			`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="currentColor" stroke-width="1" opacity=".12"/>`,
			f.left, yy, f.left+f.plotW, yy))
		b.WriteString(fmt.Sprintf(
			`<text x="%.2f" y="%.2f" text-anchor="end" font-size="11" fill="currentColor" opacity=".7">%s</text>`,
			f.left-10, yy+4, escapeText(fmtNum(v))))
	}
	b.WriteString(fmt.Sprintf(
		`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="currentColor" stroke-width="1.2" opacity=".45"/>`,
		f.left, f.top, f.left, f.top+f.plotH))
	b.WriteString(fmt.Sprintf(
		`<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="currentColor" stroke-width="1.2" opacity=".45"/>`,
		f.left, f.top+f.plotH, f.left+f.plotW, f.top+f.plotH))
	if label = strings.TrimSpace(label); label != "" {
		b.WriteString(fmt.Sprintf(
			`<text x="%.2f" y="%.2f" font-size="11" fill="currentColor" opacity=".7" transform="rotate(-90 %.2f %.2f)">%s</text>`,
			f.left-56, f.top+f.plotH/2, f.left-56, f.top+f.plotH/2, escapeText(label)))
	}
	return b.String()
}

func renderXLabel(f chartFrame, centerX float64, category string, total int) string {
	text := truncateText(category, 10)
	size := 11.0
	if total > 10 {
		size = 10
	}
	return fmt.Sprintf(
		`<text x="%.2f" y="%.2f" text-anchor="end" font-size="%.0f" fill="currentColor" opacity=".7" `+
			`transform="rotate(-28 %.2f %.2f)">%s</text>`,
		centerX, f.top+f.plotH+16, size, centerX, f.top+f.plotH+16, escapeText(text))
}

func renderXAxisTitle(f chartFrame, label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}
	return fmt.Sprintf(
		`<text x="%.2f" y="%.2f" text-anchor="middle" font-size="11" fill="currentColor" opacity=".7">%s</text>`,
		f.left+f.plotW/2, f.height-12, escapeText(label))
}

// niceScale rounds an axis to human steps (1, 2, 5 x 10^n) so tick labels are
// readable numbers rather than raw data extremes.
func niceScale(lo, hi float64) (float64, float64, float64) {
	if hi <= lo {
		hi = lo + 1
	}
	raw := (hi - lo) / 4
	exp := math.Floor(math.Log10(raw))
	frac := raw / math.Pow(10, exp)
	step := math.Pow(10, exp)
	switch {
	case frac <= 1:
		step *= 1
	case frac <= 2:
		step *= 2
	case frac <= 5:
		step *= 5
	default:
		step *= 10
	}
	if step <= 0 {
		step = 1
	}
	niceLo := math.Floor(lo/step) * step
	niceHi := math.Ceil(hi/step) * step
	return niceLo, niceHi, step
}

func chartColor(i int) string {
	return chartPalette[i%len(chartPalette)]
}

// escapeText strips control characters that would break serialisation and
// escapes the rest, so a category label can never inject markup into the SVG.
func escapeText(s string) string {
	return html.EscapeString(strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		if r < 0x20 {
			return -1
		}
		return r
	}, s))
}

func truncateText(s string, maxRunes int) string {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	if maxRunes <= 1 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-1]) + "…"
}

func fmtNum(v float64) string {
	abs := math.Abs(v)
	switch {
	case abs >= 1000:
		return strconv.FormatFloat(v, 'f', 0, 64)
	case abs >= 10:
		return strconv.FormatFloat(v, 'f', 1, 64)
	case abs > 0:
		return strconv.FormatFloat(v, 'f', 2, 64)
	default:
		return "0"
	}
}

func fmtPct(ratio float64) string {
	return strconv.FormatFloat(ratio*100, 'f', 1, 64) + "%"
}
