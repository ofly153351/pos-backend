-- Activity Center V2: capture a field-level {before, after} diff for
-- restore-eligible edits. Nullable JSONB so historical rows (and non-eligible
-- modules) simply carry NULL — the UI shows a "change details were not captured
-- before this upgrade" state rather than a fabricated diff. The payload shape is
-- {"kind":"product","fields":{"base_price":{"before":45,"after":49}}}, written by
-- the service layer via internal/platform/activitycapture and consumed by the
-- Activity Center drawer and (future) the AI Copilot's change-history source.
ALTER TABLE activity_logs ADD COLUMN IF NOT EXISTS changes JSONB;
