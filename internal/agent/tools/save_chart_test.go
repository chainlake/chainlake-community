package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// stubChartStorage captures the bytes save_chart uploads without touching a
// real backend.
type stubChartStorage struct {
	interfaces.FileService
	tenantID uint64
	name     string
	data     []byte
}

func (s *stubChartStorage) SaveBytes(
	_ context.Context, data []byte, tenantID uint64, fileName string, _ bool,
) (string, error) {
	s.tenantID = tenantID
	s.name = fileName
	s.data = data
	return "resource://chart-handle", nil
}

func chartTestContext() context.Context {
	return context.WithValue(context.Background(), types.TenantIDContextKey, uint64(7))
}

func TestSaveChartRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name string
		in   SaveChartInput
		want string
	}{
		{
			name: "empty categories",
			in:   SaveChartInput{Series: []SaveChartSeries{{Name: "a", Values: []float64{1}}}},
			want: "categories",
		},
		{
			name: "empty series",
			in:   SaveChartInput{Categories: []string{"a"}},
			want: "series",
		},
		{
			name: "unsupported type",
			in: SaveChartInput{
				ChartType:  "radar",
				Categories: []string{"a"},
				Series:     []SaveChartSeries{{Name: "s", Values: []float64{1}}},
			},
			want: "bar, line or pie",
		},
		{
			name: "pie with two series",
			in: SaveChartInput{
				ChartType:  "pie",
				Categories: []string{"a"},
				Series: []SaveChartSeries{
					{Name: "s1", Values: []float64{1}},
					{Name: "s2", Values: []float64{2}},
				},
			},
			want: "exactly one series",
		},
		{
			name: "more values than categories",
			in: SaveChartInput{
				Categories: []string{"a"},
				Series:     []SaveChartSeries{{Name: "s", Values: []float64{1, 2}}},
			},
			want: "only 1 categories",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := tc.in
			if err := validateSaveChartInput(&in); err == nil {
				t.Fatalf("expected an error, got nil")
			} else if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not mention %q", err.Error(), tc.want)
			}
		})
	}
}

func TestSaveChartRendersSelfContainedHTML(t *testing.T) {
	in := SaveChartInput{
		Title:      `销售 <趋势>`,
		ChartType:  "bar",
		Categories: []string{`<script>alert(1)</script>`, "2024", "2025"},
		Series:     []SaveChartSeries{{Name: "金额", Values: []float64{12, 30, 18}}},
		Note:       "口径：已去退款",
	}
	if err := validateSaveChartInput(&in); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	doc := renderChartHTML(&in)

	if !strings.Contains(doc, "<svg") || !strings.Contains(doc, "</svg>") {
		t.Fatal("chart document has no inline svg")
	}
	// Offline deployments must render this: no scripts, no remote assets. The
	// only URL left is the SVG namespace, which is an identifier, not a fetch.
	if strings.Contains(doc, "<script") {
		t.Fatal("chart document must not contain scripts")
	}
	for _, fetch := range []string{"src=", "href=", "@import", "url("} {
		if strings.Contains(doc, fetch) {
			t.Fatalf("chart document must not fetch anything, found %q", fetch)
		}
	}
	// A category label is data, never markup.
	if strings.Contains(doc, "<script>alert(1)</script>") {
		t.Fatal("category label was not escaped")
	}
	if !strings.Contains(doc, "&lt;script&gt;") {
		t.Fatal("escaped category label missing")
	}
	if !strings.Contains(doc, `销售 &lt;趋势&gt;`) {
		t.Fatal("title was not escaped")
	}
}

// Every chart type must survive degenerate data without emitting a broken
// coordinate: the panel renders whatever the model hands us.
func TestSaveChartGeometryStaysFinite(t *testing.T) {
	cases := []struct {
		name string
		in   SaveChartInput
	}{
		{
			name: "negative bars",
			in: SaveChartInput{
				ChartType:  "bar",
				Categories: []string{"盈利", "亏损", "持平"},
				Series:     []SaveChartSeries{{Name: "净额", Values: []float64{120, -80, 0}}},
			},
		},
		{
			name: "single line point",
			in: SaveChartInput{
				ChartType:  "line",
				Categories: []string{"唯一"},
				Series:     []SaveChartSeries{{Name: "值", Values: []float64{5}}},
			},
		},
		{
			name: "flat line",
			in: SaveChartInput{
				ChartType:  "line",
				Categories: []string{"a", "b", "c"},
				Series:     []SaveChartSeries{{Name: "值", Values: []float64{7, 7, 7}}},
			},
		},
		{
			name: "all zero pie",
			in: SaveChartInput{
				ChartType:  "pie",
				Categories: []string{"a", "b"},
				Series:     []SaveChartSeries{{Name: "占比", Values: []float64{0, 0}}},
			},
		},
		{
			name: "wide multi series",
			in: SaveChartInput{
				ChartType:  "bar",
				Categories: []string{"1", "2", "3", "4", "5", "6"},
				Series: []SaveChartSeries{
					{Name: "a", Values: []float64{1, 2, 3, 4, 5, 6}},
					{Name: "b", Values: []float64{6, 5, 4, 3, 2, 1}},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := tc.in
			if err := validateSaveChartInput(&in); err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
			doc := renderChartHTML(&in)
			// Negative coordinates are legal in SVG (rotated labels); negative
			// sizes and non-finite numbers are not.
			for _, broken := range []string{"NaN", "Inf", `width="-`, `height="-`} {
				if strings.Contains(doc, broken) {
					t.Fatalf("rendered document contains %q", broken)
				}
			}
			if strings.Count(doc, "<svg") != 1 {
				t.Fatalf("expected exactly one svg root, got %d", strings.Count(doc, "<svg"))
			}
		})
	}
}

func TestSaveChartPieRendersSingleValueAsDisc(t *testing.T) {
	in := SaveChartInput{
		Title:      "构成",
		ChartType:  "pie",
		Categories: []string{"唯一项"},
		Series:     []SaveChartSeries{{Name: "占比", Values: []float64{42}}},
	}
	if err := validateSaveChartInput(&in); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	doc := renderChartHTML(&in)
	// A 360-degree arc whose endpoints coincide draws nothing, so the renderer
	// must fall back to a filled circle.
	if !strings.Contains(doc, "<circle") {
		t.Fatal("single-slice pie should render as a disc")
	}
}

func TestSaveChartFileNameIsSafe(t *testing.T) {
	name := chartFileName(`收入/支出 (2024) "../x"`)
	if strings.ContainsAny(name, `/\`) {
		t.Fatalf("file name %q contains a path separator", name)
	}
	if !strings.HasSuffix(name, ".html") {
		t.Fatalf("file name %q is not html", name)
	}
	if strings.Contains(name, "..") {
		t.Fatalf("file name %q contains a traversal sequence", name)
	}
}

func TestSaveChartExecuteRegistersArtifact(t *testing.T) {
	storage := &stubChartStorage{}
	tool := NewSaveChartTool(storage, "session-1")

	args, err := json.Marshal(SaveChartInput{
		Title:      "月度销售额",
		ChartType:  "line",
		Categories: []string{"1月", "2月"},
		Series:     []SaveChartSeries{{Name: "销售额", Values: []float64{10, 20}}},
	})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	result, err := tool.Execute(chartTestContext(), args)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !result.Success {
		t.Fatalf("Execute failed: %s", result.Error)
	}
	if storage.tenantID != 7 {
		t.Fatalf("tenant not threaded through: got %d", storage.tenantID)
	}
	if !strings.Contains(storage.name, ".html") {
		t.Fatalf("stored name %q is not html", storage.name)
	}
	if len(result.Artifacts) != 1 {
		t.Fatalf("expected one artifact, got %d", len(result.Artifacts))
	}

	artifact := result.Artifacts[0]
	if artifact.FileType != ".html" {
		t.Fatalf("artifact type %q", artifact.FileType)
	}
	if artifact.ContentHash == "" {
		t.Fatal("artifact has no content hash")
	}
	if artifact.URL != "resource://chart-handle" {
		t.Fatalf("artifact url %q", artifact.URL)
	}
	// The model has to be told the exact name to reference, or the card never
	// appears where the answer is written.
	if !strings.Contains(result.Output, artifact.FileName) {
		t.Fatalf("output does not name the artifact: %q", result.Output)
	}
	if !strings.Contains(result.Output, "![") {
		t.Fatalf("output does not show the reference syntax: %q", result.Output)
	}
	if result.Data["display_type"] != "chart" {
		t.Fatalf("display_type = %v", result.Data["display_type"])
	}
}

func TestSaveChartWithoutStorageFailsSafely(t *testing.T) {
	tool := NewSaveChartTool(nil, "session-1")
	args, _ := json.Marshal(SaveChartInput{
		Categories: []string{"a"},
		Series:     []SaveChartSeries{{Name: "s", Values: []float64{1}}},
	})
	result, err := tool.Execute(chartTestContext(), args)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Success {
		t.Fatal("expected failure without storage")
	}
	if !strings.Contains(result.Error, "table instead") {
		t.Fatalf("error should tell the model what to do instead: %q", result.Error)
	}
}
