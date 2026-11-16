package voting

import (
	"strings"
	"testing"
)

func TestCheckRefusesWhatTheBlueprintDeclaresInvalid(t *testing.T) {
	s, err := SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	good := Poll{Question: "Which day?", Mode: "one", PerGroup: "once", State: "draft", OpensAt: "2026-11-01T08:00:00Z", ClosesAt: "2026-11-08T08:00:00Z", CreatedBy: "actor-1", CreatedAt: "2026-11-01T08:00:00Z"}
	if err := Check(s, ObjectPoll, good); err != nil {
		t.Fatalf("valid poll refused: %v", err)
	}
	if err := Check(s, ObjectPoll, &good); err != nil {
		t.Fatalf("valid poll pointer refused: %v", err)
	}
	cases := map[string]struct {
		edit func(*Poll)
		want string
	}{
		"blank question":   {func(p *Poll) { p.Question = "  " }, "question is required"},
		"undeclared mode":  {func(p *Poll) { p.Mode = "ranked" }, "mode is \"ranked\""},
		"undeclared state": {func(p *Poll) { p.State = "paused" }, "state is \"paused\""},
	}
	for name, c := range cases {
		p := good
		c.edit(&p)
		err := Check(s, ObjectPoll, p)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want it to contain %q", name, err, c.want)
		}
	}
}

func TestCheckRefusesAnUnknownObjectAndANonStruct(t *testing.T) {
	s, err := SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := Check(s, "ghost", Poll{}); err == nil || !strings.Contains(err.Error(), "fills no object") {
		t.Fatalf("unknown object err = %v", err)
	}
	if err := Check(s, ObjectPoll, "a string"); err == nil || !strings.Contains(err.Error(), "want a struct") {
		t.Fatalf("non-struct err = %v", err)
	}
}
