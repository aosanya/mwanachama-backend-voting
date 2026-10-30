package voting

import (
	"context"
	"errors"
	"testing"

	"github.com/aosanya/mwanachama-backend-shared/specstore"
)

func runOf(t *testing.T, m *pollManager, id string) *pollManager {
	t.Helper()
	r, err := m.ForRun(id)
	if err != nil {
		t.Fatal(err)
	}
	return r.(*pollManager)
}

func liveRows(t *testing.T, m *pollManager) int64 {
	t.Helper()
	var total int64
	for _, o := range m.spec.Objects {
		var n int64
		if err := m.db.Table(m.spec.TableFor(o)).Count(&n).Error; err != nil {
			t.Fatal(err)
		}
		total += n
	}
	return total
}

func testCopyRows(t *testing.T, m *pollManager) int64 {
	t.Helper()
	var total int64
	for _, o := range m.spec.Objects {
		var n int64
		if err := m.db.Table(m.spec.TestTableFor(o)).Count(&n).Error; err != nil {
			t.Fatal(err)
		}
		total += n
	}
	return total
}

func pollAndAnswer(t *testing.T, m *pollManager, by string, pick ...int) Poll {
	t.Helper()
	p := aPollWith(t, m, func(p *Poll) { p.ChangeAllowed = true })
	if _, err := answerAs(m, context.Background(), Answer{PollID: p.ID, AnsweredBy: by, Votes: votesOn(p, pick...)}, []string{"g-1"}); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestARunThroughAPollLeavesNoRowInAnyLiveTable(t *testing.T) {
	live := answeringManager(t, insideTheWindow)
	run := runOf(t, live, "run-1")
	pollAndAnswer(t, run, "a-1", 1)

	if got := liveRows(t, live); got != 0 {
		t.Fatalf("live rows = %d after a run, want 0", got)
	}
	rows, err := specstore.RowsOfRun(context.Background(), live.db, live.spec, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	stored := map[string]int64{}
	for _, r := range rows {
		stored[r.Object] = r.Rows
	}
	for _, object := range []string{objectPoll, objectChoice, objectParticipation, objectBallot} {
		if stored[object] == 0 {
			t.Fatalf("run-1 stored nothing in %s: %+v", object, rows)
		}
	}
}

func TestALiveWriteLeavesTheTestCopyEmpty(t *testing.T) {
	live := answeringManager(t, insideTheWindow)
	pollAndAnswer(t, live, "a-1", 1)

	if got := testCopyRows(t, live); got != 0 {
		t.Fatalf("test copy holds %d rows after a live write, want 0", got)
	}
}

func TestARunDoesNotSeeLiveOrOtherRunPolls(t *testing.T) {
	live := answeringManager(t, insideTheWindow)
	first, second := runOf(t, live, "run-1"), runOf(t, live, "run-2")
	livePoll := aPollWith(t, live, nil)
	firstPoll := pollAndAnswer(t, first, "a-1", 1)
	pollAndAnswer(t, second, "a-1", 0, 1)
	ctx := context.Background()

	if _, err := first.GetPoll(ctx, livePoll.ID); !errors.Is(err, ErrPollNotFound) {
		t.Fatalf("run-1 reading a live poll = %v, want ErrPollNotFound", err)
	}
	if _, err := live.GetPoll(ctx, firstPoll.ID); !errors.Is(err, ErrPollNotFound) {
		t.Fatalf("live reading run-1's poll = %v, want ErrPollNotFound", err)
	}
	if _, err := second.GetPoll(ctx, firstPoll.ID); !errors.Is(err, ErrPollNotFound) {
		t.Fatalf("run-2 reading run-1's poll = %v, want ErrPollNotFound", err)
	}
	if _, err := second.CountPoll(ctx, firstPoll.ID); !errors.Is(err, ErrPollNotFound) {
		t.Fatalf("run-2 counting run-1's poll = %v, want ErrPollNotFound", err)
	}
	c, err := first.CountPoll(ctx, firstPoll.ID)
	if err != nil || c.Answered != 1 || c.Choices[0].Votes != 1 {
		t.Fatalf("run-1 count = %+v, %v", c, err)
	}
}

func TestAChangedAnswerInARunLeavesAnotherRunsRowsAlone(t *testing.T) {
	live := answeringManager(t, insideTheWindow)
	first, second := runOf(t, live, "run-1"), runOf(t, live, "run-2")
	ctx := context.Background()
	p := pollAndAnswer(t, first, "a-1", 1)

	var mine Participation
	if err := first.st.Take(first.st.Query(ctx, objectParticipation), objectParticipation, &mine, ErrPollNotFound); err != nil {
		t.Fatal(err)
	}
	if err := second.st.Insert(ctx, objectParticipation, Participation{ID: mine.ID, PollID: "other", AnsweredBy: "a-9", GroupID: "g-1", AnsweredAt: insideTheWindow}); err != nil {
		t.Fatal(err)
	}
	if err := second.st.Insert(ctx, objectBallot, Ballot{ID: newID(), PollID: "other", ResponseID: mine.ID, GroupID: "g-1", CastAt: insideTheWindow}); err != nil {
		t.Fatal(err)
	}

	if _, err := answerAs(first, ctx, Answer{PollID: p.ID, AnsweredBy: "a-1", Votes: votesOn(p, 0, 1)}, []string{"g-1"}); err != nil {
		t.Fatal(err)
	}
	if got := countRows(t, second, objectParticipation); got != 1 {
		t.Fatalf("run-2 holds %d participations after run-1 changed an answer, want 1", got)
	}
	if got := countRows(t, second, objectBallot); got != 1 {
		t.Fatalf("run-2 holds %d ballots after run-1 changed an answer, want 1", got)
	}
}

func TestRemovingARunRemovesItsRowsAndKeepsTheOthers(t *testing.T) {
	live := answeringManager(t, insideTheWindow)
	first, second := runOf(t, live, "run-1"), runOf(t, live, "run-2")
	ctx := context.Background()
	pollAndAnswer(t, first, "a-1", 1)
	secondPoll := pollAndAnswer(t, second, "a-1", 1)

	removed, err := specstore.RemoveRun(ctx, live.db, live.spec, "run-1")
	if err != nil || removed == 0 {
		t.Fatalf("removed %d rows, %v", removed, err)
	}
	if rows, _ := specstore.RowsOfRun(ctx, live.db, live.spec, "run-1"); len(rows) != 0 {
		t.Fatalf("run-1 still holds %+v", rows)
	}
	if c, err := second.CountPoll(ctx, secondPoll.ID); err != nil || c.Answered != 1 {
		t.Fatalf("run-2 count = %+v, %v", c, err)
	}
}

func TestARunKeepsTheParentsClockAndGroups(t *testing.T) {
	live := answeringManager(t, insideTheWindow)
	live.UseGroups(func(context.Context, string) ([]string, error) { return []string{"g-1"}, nil })
	run := runOf(t, live, "run-1")
	p := aPollWith(t, run, nil)
	if _, err := run.AnswerPoll(context.Background(), Answer{PollID: p.ID, AnsweredBy: "a-1", Votes: votesOn(p, 1)}); err != nil {
		t.Fatalf("run answering with the parent's clock and groups: %v", err)
	}
}

func TestARunWithoutAnIDIsRefused(t *testing.T) {
	live := answeringManager(t, insideTheWindow)
	if _, err := live.ForRun(""); !errors.Is(err, specstore.ErrRunIDEmpty) {
		t.Fatalf("ForRun(\"\") = %v, want ErrRunIDEmpty", err)
	}
}
