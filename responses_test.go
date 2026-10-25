package voting

import (
	"context"
	"errors"
	"testing"
)

func TestAHolderReadsWhoChoseWhatAndAMemberOnlyTheirOwn(t *testing.T) {
	m := answeringManager(t, insideTheWindow)
	p := aPollWith(t, m, nil)
	ctx := context.Background()
	for i, pick := range [][]int{{1}, {0, 1}} {
		by := []string{"a-1", "a-2"}[i]
		if _, err := answerAs(m, ctx, Answer{PollID: p.ID, AnsweredBy: by, Votes: votesOn(p, pick...)}, []string{"g-1"}); err != nil {
			t.Fatalf("answer %s: %v", by, err)
		}
	}
	all, err := m.ListResponses(ctx, p.ID)
	if err != nil {
		t.Fatalf("ListResponses: %v", err)
	}
	if len(all.Items) != 2 {
		t.Fatalf("responses = %+v", all)
	}
	chose := map[string]map[string]int{}
	for _, r := range all.Items {
		chose[r.AnsweredBy] = r.Votes
	}
	if chose["a-1"][p.Choices[0].ID] != 1 || chose["a-2"][p.Choices[1].ID] != 1 || chose["a-1"][p.Choices[1].ID] != 0 {
		t.Fatalf("who chose what = %+v", chose)
	}
	mine, err := m.ReadOwnResponses(ctx, p.ID, "a-2")
	if err != nil {
		t.Fatalf("ReadOwnResponses: %v", err)
	}
	if len(mine.Items) != 1 || mine.Items[0].AnsweredBy != "a-2" || mine.Items[0].Votes[p.Choices[1].ID] != 1 {
		t.Fatalf("own = %+v", mine)
	}
	none, err := m.ReadOwnResponses(ctx, p.ID, "a-9")
	if err != nil || len(none.Items) != 0 {
		t.Fatalf("a member who did not answer reads %+v, %v", none, err)
	}
	if _, err := m.ReadOwnResponses(ctx, p.ID, ""); !errors.Is(err, ErrNoReader) {
		t.Fatalf("no caller: %v", err)
	}
	if _, err := m.ListResponses(ctx, "missing"); !errors.Is(err, ErrPollNotFound) {
		t.Fatalf("missing poll: %v", err)
	}
}
