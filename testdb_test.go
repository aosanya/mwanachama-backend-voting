package voting_test

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	voting "github.com/aosanya/mwanachama-backend-voting"
)

func openDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return db
}

func newTestManager(t *testing.T) voting.PollManager {
	t.Helper()
	db := openDB(t)
	s, err := voting.SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := voting.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	pm, err := voting.NewPollManager(db, s)
	if err != nil {
		t.Fatalf("NewPollManager: %v", err)
	}
	return pm
}

func aPoll() voting.Poll {
	return voting.Poll{
		Question:  "Which day suits the meeting?",
		Mode:      "one",
		GroupIDs:  []string{"g-1"},
		OpensAt:   "2026-11-01T08:00:00Z",
		ClosesAt:  "2026-11-08T08:00:00Z",
		CreatedBy: "actor-1",
	}
}

func TestADraftPollKeepsItsChoicesInOrder(t *testing.T) {
	ctx := context.Background()
	pm := newTestManager(t)
	p, err := pm.CreatePoll(ctx, aPoll())
	if err != nil {
		t.Fatalf("CreatePoll: %v", err)
	}
	if p.State != "draft" || p.PerGroup != "once" {
		t.Fatalf("new poll = %+v", p)
	}
	for _, l := range []string{"Monday", "Tuesday"} {
		if _, err := pm.AddChoice(ctx, voting.Choice{PollID: p.ID, Label: l}); err != nil {
			t.Fatalf("AddChoice: %v", err)
		}
	}
	cs, err := pm.ListChoices(ctx, p.ID)
	if err != nil || len(cs) != 2 || cs[0].Label != "Monday" || cs[1].Position != 2 {
		t.Fatalf("ListChoices = %+v, %v", cs, err)
	}
	got, err := pm.GetPoll(ctx, p.ID)
	if err != nil || len(got.GroupIDs) != 1 || got.ClosesAt != p.ClosesAt {
		t.Fatalf("GetPoll = %+v, %v", got, err)
	}
}

func TestAPollWithNoCallerIsRefused(t *testing.T) {
	in := aPoll()
	in.CreatedBy = ""
	if _, err := newTestManager(t).CreatePoll(context.Background(), in); !errors.Is(err, voting.ErrNoCreator) {
		t.Fatalf("err = %v, want ErrNoCreator", err)
	}
}

func TestSaveRefusesAPollWhoseSettingsCannotHold(t *testing.T) {
	pm := newTestManager(t)
	cases := map[string]func(*voting.Poll){
		"backwards window": func(p *voting.Poll) { p.ClosesAt = p.OpensAt },
		"no group":         func(p *voting.Poll) { p.GroupIDs = nil },
		"up_to with no N":  func(p *voting.Poll) { p.Mode = "up_to" },
		"no budget":        func(p *voting.Poll) { p.Mode = "quadratic" },
		"undeclared mode":  func(p *voting.Poll) { p.Mode = "ranked" },
	}
	for name, spoil := range cases {
		in := aPoll()
		spoil(&in)
		if _, err := pm.CreatePoll(context.Background(), in); !errors.Is(err, voting.ErrInvalidPoll) {
			t.Errorf("%s: err = %v, want ErrInvalidPoll", name, err)
		}
	}
}

func TestAChoiceNeedsAPollThatExists(t *testing.T) {
	_, err := newTestManager(t).AddChoice(context.Background(), voting.Choice{PollID: "nope", Label: "x"})
	if !errors.Is(err, voting.ErrPollNotFound) {
		t.Fatalf("err = %v, want ErrPollNotFound", err)
	}
}

func TestAPollSavedWithItsChoicesKeepsThemInOrder(t *testing.T) {
	ctx := context.Background()
	pm := newTestManager(t)
	in := aPoll()
	in.Mode, in.MaxChoices = "up_to", 2
	in.Choices = []voting.Choice{{Label: "Monday", PollID: "elsewhere", ID: "forged"}, {Label: "Tuesday"}, {Label: "Friday"}}
	p, err := pm.CreatePoll(ctx, in)
	if err != nil {
		t.Fatalf("CreatePoll: %v", err)
	}
	cs, err := pm.ListChoices(ctx, p.ID)
	if err != nil || len(cs) != 3 || cs[0].Label != "Monday" || cs[0].ID == "forged" || cs[2].Position != 3 {
		t.Fatalf("ListChoices = %+v, %v", cs, err)
	}
	if len(p.Choices) != 3 || p.Choices[0].PollID != p.ID {
		t.Fatalf("CreatePoll choices = %+v", p.Choices)
	}
}

func TestSaveRefusesAnUpToOverItsChoices(t *testing.T) {
	ctx := context.Background()
	pm := newTestManager(t)
	in := aPoll()
	in.Mode, in.MaxChoices = "up_to", 3
	in.Choices = []voting.Choice{{Label: "Monday"}, {Label: "Tuesday"}}
	if _, err := pm.CreatePoll(ctx, in); !errors.Is(err, voting.ErrInvalidPoll) {
		t.Fatalf("err = %v, want ErrInvalidPoll", err)
	}
}

func TestSaveRefusesAChoiceWithNoLabel(t *testing.T) {
	in := aPoll()
	in.Choices = []voting.Choice{{Label: "Monday"}, {Label: " "}}
	if _, err := newTestManager(t).CreatePoll(context.Background(), in); !errors.Is(err, voting.ErrInvalidChoice) {
		t.Fatalf("err = %v, want ErrInvalidChoice", err)
	}
}
