// Package activitycapture lets the service layer record a field-level {before,
// after} diff for an entity edit, which the activity-log middleware then attaches
// to the audit row after the handler returns.
//
// Flow: the ActivityLog middleware installs a Recorder into the request context
// (c.UserContext) before c.Next(). Handlers already pass c.UserContext() to their
// services, so a service's Update method can call Record(ctx, kind, before, after)
// at its mutation point. After c.Next() the middleware reads Recorder.JSON() and
// stores it. Calls are no-ops when no recorder is present (e.g. background jobs),
// so services never need to guard for it.
package activitycapture

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
)

// FieldChange is a single field's previous and new value, using the entity's API
// field names (e.g. "base_price") so the diff is restore- and display-ready.
type FieldChange struct {
	Before any `json:"before"`
	After  any `json:"after"`
}

// payload is the serialized shape stored in activity_logs.changes.
type payload struct {
	Kind   string                 `json:"kind"`
	Fields map[string]FieldChange `json:"fields"`
}

// Recorder accumulates the changed fields for one request. Safe for concurrent use.
type Recorder struct {
	mu     sync.Mutex
	kind   string
	fields map[string]FieldChange
}

func New() *Recorder { return &Recorder{} }

// SetKind records the entity type ("product", "promotion", "store",
// "receipt_settings", "member"). Restore dispatches on this, since the middleware
// canonicalises several entities under the same module name.
func (r *Recorder) SetKind(kind string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.kind = kind
	r.mu.Unlock()
}

// Set records field's before/after only if they actually differ. nil-safe.
func (r *Recorder) Set(field string, before, after any) {
	if r == nil || equalValues(before, after) {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fields == nil {
		r.fields = make(map[string]FieldChange)
	}
	r.fields[field] = FieldChange{Before: before, After: after}
}

// HasChanges reports whether at least one field changed.
func (r *Recorder) HasChanges() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.fields) > 0
}

// JSON returns the serialized {kind, fields} payload, or nil when nothing changed.
func (r *Recorder) JSON() []byte {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.fields) == 0 {
		return nil
	}
	b, err := json.Marshal(payload{Kind: r.kind, Fields: r.fields})
	if err != nil {
		return nil
	}
	return b
}

type ctxKey struct{}

// WithRecorder attaches a recorder to ctx (called by the middleware before c.Next).
func WithRecorder(ctx context.Context, r *Recorder) context.Context {
	return context.WithValue(ctx, ctxKey{}, r)
}

// FromContext returns the recorder, or nil if none is installed.
func FromContext(ctx context.Context) *Recorder {
	r, _ := ctx.Value(ctxKey{}).(*Recorder)
	return r
}

// Record is the ergonomic entry point for services: pass the entity kind and two
// snapshots (field name → value) taken before and after the mutation. Only fields
// that changed are stored. No-op when no recorder is in ctx.
func Record(ctx context.Context, kind string, before, after map[string]any) {
	r := FromContext(ctx)
	if r == nil {
		return
	}
	r.SetKind(kind)
	seen := make(map[string]struct{}, len(before)+len(after))
	for k := range before {
		seen[k] = struct{}{}
	}
	for k := range after {
		seen[k] = struct{}{}
	}
	for k := range seen {
		r.Set(k, before[k], after[k])
	}
}

// equalValues compares two field values, transparently dereferencing pointers so a
// *string and the same string compare equal.
func equalValues(a, b any) bool {
	return reflect.DeepEqual(deref(a), deref(b))
}

func deref(v any) any {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		return rv.Elem().Interface()
	}
	return v
}
