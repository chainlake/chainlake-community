package agent

import "github.com/Tencent/WeKnora/internal/types"

// mergeTurnArtifacts joins artifact lists produced within one turn — delegated
// work and tools that persisted output themselves — into the single list the
// completion event carries.
//
// Deduplication is by URL, falling back to the file name when a deployment
// without a resource catalog stores a bare path. Tombstoned entries are dropped
// so a deleted artifact cannot come back through a later merge.
func mergeTurnArtifacts(lists ...types.MessageArtifacts) types.MessageArtifacts {
	total := 0
	for _, list := range lists {
		total += len(list)
	}
	if total == 0 {
		return nil
	}

	seen := make(map[string]struct{}, total)
	out := make(types.MessageArtifacts, 0, total)
	for _, list := range lists {
		for _, artifact := range list.Live() {
			key := artifact.URL
			if key == "" {
				key = artifact.FileName
			}
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, artifact)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
