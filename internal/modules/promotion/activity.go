package promotion

import (
	"context"
	"encoding/json"
	"strings"

	"pos-backend/internal/platform/activitycapture"
)

// recordPromotionChange logs a {before, after} diff for a promotion edit. The
// scalar meta fields (name/type/status/code) drive the Activity Center timeline
// display; the full campaign object is stored under "data" so the restore flow can
// re-apply the previous configuration verbatim (the frontend skips object-valued
// fields when rendering the human diff).
func recordPromotionChange(ctx context.Context, oldData, newData string) {
	if strings.TrimSpace(oldData) == "" {
		// Couldn't load the prior state — skip rather than show a misleading
		// "everything is new" diff.
		return
	}
	activitycapture.Record(ctx, "promotion", promotionSnapshot(oldData), promotionSnapshot(newData))
}

func promotionSnapshot(data string) map[string]any {
	meta := extractMeta([]byte(data))
	snap := map[string]any{
		"name":   meta.Name,
		"type":   meta.Type,
		"status": meta.Status,
		"code":   meta.Code,
	}
	var obj map[string]any
	if json.Unmarshal([]byte(data), &obj) == nil {
		snap["data"] = obj
	}
	return snap
}
