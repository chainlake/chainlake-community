package agent

import (
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestMergeTurnArtifacts(t *testing.T) {
	deletedAt := time.Now()
	collab := types.MessageArtifacts{
		{URL: "resource://a", FileName: "a.html"},
		{URL: "resource://gone", FileName: "gone.html", DeletedAt: &deletedAt},
	}
	tool := types.MessageArtifacts{
		{URL: "resource://a", FileName: "a.html"}, // same file, later source
		{URL: "resource://b", FileName: "b.html"},
	}

	merged := mergeTurnArtifacts(collab, tool)

	if len(merged) != 2 {
		t.Fatalf("expected 2 artifacts, got %d: %+v", len(merged), merged)
	}
	if merged[0].URL != "resource://a" || merged[1].URL != "resource://b" {
		t.Fatalf("unexpected order/content: %+v", merged)
	}
	for _, artifact := range merged {
		if artifact.Deleted() {
			t.Fatalf("tombstone survived the merge: %s", artifact.FileName)
		}
	}
}

func TestMergeTurnArtifactsFallsBackToFileName(t *testing.T) {
	// Deployments without a resource catalog store a bare path, so identity
	// has to come from somewhere else.
	merged := mergeTurnArtifacts(
		types.MessageArtifacts{{FileName: "chart.html"}},
		types.MessageArtifacts{{FileName: "chart.html"}},
	)
	if len(merged) != 1 {
		t.Fatalf("expected dedupe by file name, got %d", len(merged))
	}
}

func TestMergeTurnArtifactsEmpty(t *testing.T) {
	if merged := mergeTurnArtifacts(nil, types.MessageArtifacts{}); merged != nil {
		t.Fatalf("expected nil for empty input, got %+v", merged)
	}
}
