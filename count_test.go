package voting

import (
	"context"
	"errors"
	"testing"
)

func votesPerChoice(c Count) []int {
	out := make([]int, len(c.Choices))
	for i, cc := range c.Choices {
		out[i] = cc.Votes
	}
	return out
}

func TestACountSumsVotesPerChoiceAndCountsWhoAnswered(t *testing.T) {
	m := answeringManager(t, insideTheWindow)
	p := aPollWith(t, m, nil)
	ctx := context.Background()
	for i, pick := range [][]int{{1}, {0, 1}, {1}} {
		by := []string{"a-1", "a-2", "a-3"}[i]
		if _, err := answerAs(m, ctx, Answer{PollID: p.ID, AnsweredBy: by, Votes: votesOn(p, pick...)}, []string{"g-1"}); err != nil {
			t.Fatalf("answer %s: %v", by, err)
		}
	}
	c, err := m.CountPoll(ctx, p.ID)
	if err != nil {
		t.Fatalf("CountPoll: %v", err)
	}
	if got := votesPerChoice(c); got[0] != 2 || got[1] != 1 || got[2] != 0 {
		t.Fatalf("votes = %v, want [2 1 0]", got)
	}
	if c.Answered != 3 || c.Choices[0].Label != "Monday" || c.Choices[2].Position != 3 {
		t.Fatalf("count = %+v", c)
	}
}

func TestAQuadraticCountSumsTheVotesEachChoiceReceived(t *testing.T) {
	m := answeringManager(t, insideTheWindow)
	p := aPollWith(t, m, func(p *Poll) { p.Mode = "quadratic"; p.Budget = 9 })
	ctx := context.Background()
	if _, err := answerAs(m, ctx, Answer{PollID: p.ID, AnsweredBy: "a-1", Votes: votesOn(p, 3)}, []string{"g-1"}); err != nil {
		t.Fatalf("a-1: %v", err)
	}
	if _, err := answerAs(m, ctx, Answer{PollID: p.ID, AnsweredBy: "a-2", Votes: votesOn(p, 2, 2)}, []string{"g-1"}); err != nil {
		t.Fatalf("a-2: %v", err)
	}
	c, err := m.CountPoll(ctx, p.ID)
	if err != nil {
		t.Fatalf("CountPoll: %v", err)
	}
	if got := votesPerChoice(c); got[0] != 5 || got[1] != 2 || got[2] != 0 || c.Answered != 2 {
		t.Fatalf("count = %+v", c)
	}
}

func TestAChangedAnswerLeavesNoStaleTotal(t *testing.T) {
	m := answeringManager(t, insideTheWindow)
	p := aPollWith(t, m, func(p *Poll) { p.ChangeAllowed = true })
	ctx := context.Background()
	for _, pick := range [][]int{{1}, {0, 0, 1}} {
		if _, err := answerAs(m, ctx, Answer{PollID: p.ID, AnsweredBy: "a-1", Votes: votesOn(p, pick...)}, []string{"g-1"}); err != nil {
			t.Fatalf("answer: %v", err)
		}
	}
	c, err := m.CountPoll(ctx, p.ID)
	if err != nil {
		t.Fatalf("CountPoll: %v", err)
	}
	if got := votesPerChoice(c); got[0] != 0 || got[2] != 1 || c.Answered != 1 {
		t.Fatalf("count = %+v", c)
	}
}

func TestPerGroupCountsEachMemberAndGroupAsOneAnswer(t *testing.T) {
	m := answeringManager(t, insideTheWindow)
	p := aPollWith(t, m, func(p *Poll) { p.PerGroup = "per_group" })
	ctx := context.Background()
	for _, g := range []string{"g-1", "g-2"} {
		if _, err := answerAs(m, ctx, Answer{PollID: p.ID, AnsweredBy: "a-1", GroupID: g, Votes: votesOn(p, 1)}, []string{"g-1", "g-2"}); err != nil {
			t.Fatalf("answer for %s: %v", g, err)
		}
	}
	c, err := m.CountPoll(ctx, p.ID)
	if err != nil {
		t.Fatalf("CountPoll: %v", err)
	}
	if c.Answered != 2 || c.Choices[0].Votes != 2 {
		t.Fatalf("count = %+v", c)
	}
}

func TestACountOfAMissingPollIsNotFound(t *testing.T) {
	m := answeringManager(t, insideTheWindow)
	if _, err := m.CountPoll(context.Background(), "nope"); !errors.Is(err, ErrPollNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestANewPollCountsZero(t *testing.T) {
	m := answeringManager(t, insideTheWindow)
	p := aPollWith(t, m, nil)
	c, err := m.CountPoll(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("CountPoll: %v", err)
	}
	if len(c.Choices) != 3 || c.Answered != 0 || c.Choices[0].Votes != 0 {
		t.Fatalf("count = %+v", c)
	}
}
