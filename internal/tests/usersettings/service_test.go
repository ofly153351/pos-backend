package usersettings_test

import (
	"context"
	"testing"

	"pos-backend/internal/modules/usersettings"
)

// fakeRepo is an in-memory Repository for service tests.
type fakeRepo struct {
	store map[string]usersettings.CardSettings
	saveErr error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{store: map[string]usersettings.CardSettings{}}
}

func (r *fakeRepo) GetCardSettings(_ context.Context, userID string) (usersettings.CardSettings, bool, error) {
	cs, ok := r.store[userID]
	return cs, ok, nil
}

func (r *fakeRepo) SaveCardSettings(_ context.Context, userID string, settings usersettings.CardSettings) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.store[userID] = settings
	return nil
}

func TestGet_ReturnsDefaultsWhenUnset(t *testing.T) {
	svc := usersettings.NewService(newFakeRepo())
	got, err := svc.GetCardSettings(context.Background(), "usr-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := usersettings.DefaultCardSettings()
	if got != want {
		t.Errorf("expected defaults %+v, got %+v", want, got)
	}
}

func TestGet_RequiresUserID(t *testing.T) {
	svc := usersettings.NewService(newFakeRepo())
	if _, err := svc.GetCardSettings(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty user id")
	}
}

func TestSave_PersistsAndReturnsNormalized(t *testing.T) {
	repo := newFakeRepo()
	svc := usersettings.NewService(repo)

	in := usersettings.CardSettings{
		NamePos: "top", Fit: "contain", Aspect: "4/3", Lines: 3, Size: "lg", ShowStock: false,
	}
	got, err := svc.SaveCardSettings(context.Background(), "usr-1", in)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if got != in {
		t.Errorf("expected returned settings %+v, got %+v", in, got)
	}
	// Round-trip through Get.
	fetched, _ := svc.GetCardSettings(context.Background(), "usr-1")
	if fetched != in {
		t.Errorf("expected persisted %+v, got %+v", in, fetched)
	}
}

func TestSave_NormalizesInvalidValues(t *testing.T) {
	svc := usersettings.NewService(newFakeRepo())
	in := usersettings.CardSettings{
		NamePos: "sideways", Fit: "stretch", Aspect: "16/9", Lines: 9, Size: "xl", ShowStock: true,
	}
	got, err := svc.SaveCardSettings(context.Background(), "usr-1", in)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}
	d := usersettings.DefaultCardSettings()
	if got.NamePos != d.NamePos || got.Fit != d.Fit || got.Aspect != d.Aspect || got.Lines != d.Lines || got.Size != d.Size {
		t.Errorf("invalid values not normalized to defaults: %+v", got)
	}
}

func TestSave_RequiresUserID(t *testing.T) {
	svc := usersettings.NewService(newFakeRepo())
	if _, err := svc.SaveCardSettings(context.Background(), "", usersettings.DefaultCardSettings()); err == nil {
		t.Fatal("expected error for empty user id")
	}
}
