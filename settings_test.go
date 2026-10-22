package voting_test

import (
	"context"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/spec"

	voting "github.com/aosanya/mwanachama-backend-voting"
)

func TestTheBlueprintsSettingsDefaultsPassTheirOwnCheck(t *testing.T) {
	b, err := voting.Blueprint()
	if err != nil {
		t.Fatal(err)
	}
	if p := spec.CheckSettings(b.Settings, map[string]any{"mode": "quadratic", "budget": 100}); len(p) != 0 {
		t.Fatalf("problems: %v", p)
	}
	if p := spec.CheckSettings(b.Settings, map[string]any{"mode": "ranked"}); len(p) == 0 {
		t.Fatal("an undeclared mode passed")
	}
}

func TestANewPollStartsFromTheEffectiveSettings(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()
	s, err := m.Settings(ctx)
	if err != nil || s.Mode != "one" || s.PerGroup != "once" {
		t.Fatalf("module defaults = %+v, %v", s, err)
	}
	m.UseSettings(func(context.Context) (map[string]any, map[string]spec.Override, error) {
		return map[string]any{"mode": "up_to", "max_choices": 2},
			map[string]spec.Override{"mode": {Value: "quadratic", By: "owner-1", At: "2026-10-22T20:00:00Z"}, "budget": {Value: 9, By: "owner-1", At: "2026-10-22T20:00:00Z"}}, nil
	})
	in := aPoll()
	in.Mode = ""
	in.Choices = []voting.Choice{{Label: "Monday"}, {Label: "Tuesday"}}
	p, err := m.CreatePoll(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if p.Mode != "quadratic" || p.Budget != 9 || p.PerGroup != "once" {
		t.Fatalf("poll = %+v", p)
	}
}
