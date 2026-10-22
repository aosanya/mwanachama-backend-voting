package voting

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func answeringManager(t *testing.T, now string) *pollManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := SpecFor("test")
	if err != nil {
		t.Fatalf("SpecFor: %v", err)
	}
	if err := Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	pm, err := NewPollManager(db, s)
	if err != nil {
		t.Fatalf("NewPollManager: %v", err)
	}
	m := pm.(*pollManager)
	m.now = func() string { return now }
	return m
}

const insideTheWindow = "2026-11-02T08:00:00.000000000Z"

func aPollWith(t *testing.T, m *pollManager, edit func(*Poll)) Poll {
	t.Helper()
	p := Poll{
		Question:  "Which day suits the meeting?",
		Mode:      "one",
		GroupIDs:  []string{"g-1", "g-2"},
		OpensAt:   "2026-11-01T08:00:00Z",
		ClosesAt:  "2026-11-08T08:00:00Z",
		CreatedBy: "actor-1",
		Choices:   []Choice{{Label: "Monday"}, {Label: "Tuesday"}, {Label: "Wednesday"}},
	}
	if edit != nil {
		edit(&p)
	}
	out, err := m.CreatePoll(context.Background(), p)
	if err != nil {
		t.Fatalf("CreatePoll: %v", err)
	}
	return out
}

func votesOn(p Poll, counts ...int) map[string]int {
	out := map[string]int{}
	for i, n := range counts {
		if n > 0 {
			out[p.Choices[i].ID] = n
		}
	}
	return out
}

func countRows(t *testing.T, m *pollManager, role string) int64 {
	t.Helper()
	var n int64
	if err := m.st.Query(context.Background(), role).Count(&n).Error; err != nil {
		t.Fatalf("count %s: %v", role, err)
	}
	return n
}

func TestAnAnswerIsTakenOnlyInsideTheWindow(t *testing.T) {
	for _, now := range []string{"2026-10-31T08:00:00.000000000Z", "2026-11-08T08:00:00.000000000Z"} {
		m := answeringManager(t, now)
		p := aPollWith(t, m, nil)
		_, err := answerAs(m, context.Background(), Answer{PollID: p.ID, AnsweredBy: "a-1", Votes: votesOn(p, 1)}, []string{"g-1"})
		if !errors.Is(err, ErrOutsideWindow) {
			t.Fatalf("at %s err = %v, want ErrOutsideWindow", now, err)
		}
	}
	m := answeringManager(t, "2026-11-01T08:00:00.000000000Z")
	p := aPollWith(t, m, nil)
	if _, err := answerAs(m, context.Background(), Answer{PollID: p.ID, AnsweredBy: "a-1", Votes: votesOn(p, 1)}, []string{"g-1"}); err != nil {
		t.Fatalf("at the opening instant: %v", err)
	}
}

func TestAnAnswerNeedsACallerInOneOfThePollsGroups(t *testing.T) {
	m := answeringManager(t, insideTheWindow)
	p := aPollWith(t, m, nil)
	ctx := context.Background()
	if _, err := answerAs(m, ctx, Answer{PollID: p.ID, Votes: votesOn(p, 1)}, []string{"g-1"}); !errors.Is(err, ErrNoAnswerer) {
		t.Fatalf("no caller err = %v", err)
	}
	if _, err := answerAs(m, ctx, Answer{PollID: p.ID, AnsweredBy: "a-1", Votes: votesOn(p, 1)}, []string{"g-9"}); !errors.Is(err, ErrNotInPollGroups) {
		t.Fatalf("outsider err = %v", err)
	}
	part, err := answerAs(m, ctx, Answer{PollID: p.ID, AnsweredBy: "a-1", GroupID: "g-9", Votes: votesOn(p, 1)}, []string{"g-2"})
	if err != nil {
		t.Fatalf("member: %v", err)
	}
	if part.AnsweredBy != "a-1" || part.GroupID != "" {
		t.Fatalf("participation = %+v", part)
	}
}

func TestTheVotesFitThePollsMode(t *testing.T) {
	cases := []struct {
		name  string
		edit  func(*Poll)
		votes []int
		ok    bool
	}{
		{"one takes one", nil, []int{1}, true},
		{"one refuses two", nil, []int{1, 1}, false},
		{"one refuses two votes on a choice", nil, []int{2}, false},
		{"up_to takes N", func(p *Poll) { p.Mode, p.MaxChoices = "up_to", 2 }, []int{1, 1}, true},
		{"up_to refuses more than N", func(p *Poll) { p.Mode, p.MaxChoices = "up_to", 2 }, []int{1, 1, 1}, false},
		{"quadratic takes 2 and 2 in 8", func(p *Poll) { p.Mode, p.Budget = "quadratic", 8 }, []int{2, 2}, true},
		{"quadratic refuses 3 in 8", func(p *Poll) { p.Mode, p.Budget = "quadratic", 8 }, []int{3}, false},
		{"nothing chosen", nil, []int{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := answeringManager(t, insideTheWindow)
			p := aPollWith(t, m, c.edit)
			_, err := answerAs(m, context.Background(), Answer{PollID: p.ID, AnsweredBy: "a-1", Votes: votesOn(p, c.votes...)}, []string{"g-1"})
			if c.ok && err != nil {
				t.Fatalf("err = %v", err)
			}
			if !c.ok && !errors.Is(err, ErrInvalidAnswer) {
				t.Fatalf("err = %v, want ErrInvalidAnswer", err)
			}
		})
	}
	m := answeringManager(t, insideTheWindow)
	p := aPollWith(t, m, nil)
	if _, err := answerAs(m, context.Background(), Answer{PollID: p.ID, AnsweredBy: "a-1", Votes: map[string]int{"elsewhere": 1}}, []string{"g-1"}); !errors.Is(err, ErrInvalidAnswer) {
		t.Fatalf("unknown choice err = %v", err)
	}
}

func TestASecondAnswerIsRefusedOrReplacesTheFirst(t *testing.T) {
	ctx := context.Background()
	m := answeringManager(t, insideTheWindow)
	p := aPollWith(t, m, nil)
	if _, err := answerAs(m, ctx, Answer{PollID: p.ID, AnsweredBy: "a-1", Votes: votesOn(p, 1)}, []string{"g-1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := answerAs(m, ctx, Answer{PollID: p.ID, AnsweredBy: "a-1", Votes: votesOn(p, 0, 1)}, []string{"g-1"}); !errors.Is(err, ErrAlreadyAnswered) {
		t.Fatalf("second answer err = %v", err)
	}

	m = answeringManager(t, insideTheWindow)
	p = aPollWith(t, m, func(p *Poll) { p.ChangeAllowed = true })
	first, err := answerAs(m, ctx, Answer{PollID: p.ID, AnsweredBy: "a-1", Votes: votesOn(p, 1)}, []string{"g-1"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := answerAs(m, ctx, Answer{PollID: p.ID, AnsweredBy: "a-1", Votes: votesOn(p, 0, 1)}, []string{"g-1"})
	if err != nil {
		t.Fatalf("change: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("response id moved from %s to %s", first.ID, second.ID)
	}
	if countRows(t, m, objectParticipation) != 1 || countRows(t, m, objectBallot) != 1 {
		t.Fatalf("a replaced answer left %d participations and %d ballots",
			countRows(t, m, objectParticipation), countRows(t, m, objectBallot))
	}
	var b Ballot
	if err := m.st.Take(m.st.Query(ctx, objectBallot), objectBallot, &b, ErrPollNotFound); err != nil {
		t.Fatal(err)
	}
	if b.Votes[p.Choices[1].ID] != 1 || len(b.Votes) != 1 || b.ResponseID != first.ID {
		t.Fatalf("ballot = %+v", b)
	}
}

func TestPerGroupTakesOneAnswerForEachOfTheCallersGroups(t *testing.T) {
	ctx := context.Background()
	m := answeringManager(t, insideTheWindow)
	p := aPollWith(t, m, func(p *Poll) { p.PerGroup = "per_group" })
	mine := []string{"g-1", "g-2"}
	if _, err := answerAs(m, ctx, Answer{PollID: p.ID, AnsweredBy: "a-1", Votes: votesOn(p, 1)}, mine); !errors.Is(err, ErrNotInPollGroups) {
		t.Fatalf("no group named err = %v", err)
	}
	if _, err := answerAs(m, ctx, Answer{PollID: p.ID, AnsweredBy: "a-1", GroupID: "g-2", Votes: votesOn(p, 1)}, []string{"g-1"}); !errors.Is(err, ErrNotInPollGroups) {
		t.Fatalf("a group the caller is not in err = %v", err)
	}
	for _, g := range mine {
		part, err := answerAs(m, ctx, Answer{PollID: p.ID, AnsweredBy: "a-1", GroupID: g, Votes: votesOn(p, 1)}, mine)
		if err != nil {
			t.Fatalf("group %s: %v", g, err)
		}
		if part.GroupID != g {
			t.Fatalf("participation = %+v", part)
		}
	}
	if _, err := answerAs(m, ctx, Answer{PollID: p.ID, AnsweredBy: "a-1", GroupID: "g-1", Votes: votesOn(p, 1)}, mine); !errors.Is(err, ErrAlreadyAnswered) {
		t.Fatalf("second answer for g-1 err = %v", err)
	}
}

func answerAs(m *pollManager, ctx context.Context, in Answer, groups []string) (Participation, error) {
	m.UseGroups(func(context.Context, string) ([]string, error) { return groups, nil })
	return m.AnswerPoll(ctx, in)
}

func TestAManagerGivenNoGroupsTakesNoAnswer(t *testing.T) {
	m := answeringManager(t, insideTheWindow)
	p := aPollWith(t, m, nil)
	_, err := m.AnswerPoll(context.Background(), Answer{PollID: p.ID, AnsweredBy: "a-1", Votes: votesOn(p, 1)})
	if !errors.Is(err, ErrNotInPollGroups) {
		t.Fatalf("err = %v, want ErrNotInPollGroups", err)
	}
}
