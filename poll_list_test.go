package voting

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

func openPollIDs(list []OpenPoll) []string {
	ids := make([]string, 0, len(list))
	for _, o := range list {
		ids = append(ids, o.ID)
	}
	slices.Sort(ids)
	return ids
}

func (f *pollsFixture) answer(t *testing.T, p Poll, caller string) {
	t.Helper()
	_, err := f.pm.AnswerPoll(context.Background(), Answer{PollID: p.ID, AnsweredBy: caller, Votes: map[string]int{p.Choices[0].ID: 1}})
	if err != nil {
		t.Fatalf("AnswerPoll: %v", err)
	}
}

func TestAnOpenPollIsListedWithItsChoicesInOrderAndNotAnsweredYet(t *testing.T) {
	f := newPolls(t)
	p := f.poll(t, nil)
	got, err := f.pm.ListOpenPolls(context.Background(), "amina")
	if err != nil {
		t.Fatalf("ListOpenPolls: %v", err)
	}
	if len(got) != 1 || got[0].ID != p.ID || got[0].Question != p.Question {
		t.Fatalf("list = %+v", got)
	}
	labels := []string{}
	for _, c := range got[0].Choices {
		labels = append(labels, c.Label)
	}
	if !slices.Equal(labels, []string{"Mill Street", "Orchard Lane", "Mill Road"}) {
		t.Fatalf("choices = %v", labels)
	}
	if got[0].Answered {
		t.Fatalf("a poll nobody answered is listed as answered: %+v", got[0])
	}
}

func TestTheSamePollIsAnsweredOnceTheCallerHasAnswered(t *testing.T) {
	f := newPolls(t)
	p := f.poll(t, func(p *Poll) { p.GroupIDs = []string{"ward-a"} })
	f.answer(t, p, "amina")
	ctx := context.Background()
	mine, err := f.pm.ListOpenPolls(ctx, "amina")
	if err != nil {
		t.Fatal(err)
	}
	if len(mine) != 1 || !mine[0].Answered {
		t.Fatalf("after answering = %+v", mine)
	}
	other, err := f.pm.ListOpenPolls(ctx, "otieno")
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 1 || other[0].Answered {
		t.Fatalf("another member of the group sees the answer as theirs: %+v", other)
	}
}

func TestAnsweredComesFromTheParticipationRecordNotTheBallot(t *testing.T) {
	f := newPolls(t)
	p := f.poll(t, nil)
	f.answer(t, p, "amina")
	if err := f.pm.st.Query(context.Background(), objectBallot).Where("poll_id = ?", p.ID).Delete(nil).Error; err != nil {
		t.Fatalf("delete ballots: %v", err)
	}
	got, err := f.pm.ListOpenPolls(context.Background(), "amina")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].Answered {
		t.Fatalf("answered depended on the ballot: %+v", got)
	}
}

func TestOnlyPollsInsideTheirWindowForTheCallersGroupsAreListed(t *testing.T) {
	f := newPolls(t)
	open := f.poll(t, nil)
	f.poll(t, func(p *Poll) {
		p.OpensAt = f.now.Add(time.Hour).Format(time.RFC3339)
		p.ClosesAt = f.now.Add(2 * time.Hour).Format(time.RFC3339)
	})
	f.poll(t, func(p *Poll) {
		p.OpensAt = f.now.Add(-2 * time.Hour).Format(time.RFC3339)
		p.ClosesAt = f.now.Add(-time.Hour).Format(time.RFC3339)
	})
	f.poll(t, func(p *Poll) { p.GroupIDs = []string{"ward-b"} })
	got, err := f.pm.ListOpenPolls(context.Background(), "amina")
	if err != nil {
		t.Fatal(err)
	}
	if ids := openPollIDs(got); !slices.Equal(ids, []string{open.ID}) {
		t.Fatalf("listed %v, want only %s", ids, open.ID)
	}
}

func TestACallerInSeveralGroupsSeesEachOpenPollOnce(t *testing.T) {
	f := newPolls(t)
	a := f.poll(t, func(p *Poll) { p.GroupIDs = []string{"ward-a"} })
	b := f.poll(t, func(p *Poll) { p.GroupIDs = []string{"ward-b"} })
	both := f.poll(t, func(p *Poll) { p.GroupIDs = []string{"ward-a", "ward-b"} })
	got, err := f.pm.ListOpenPolls(context.Background(), "otieno")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{a.ID, b.ID, both.ID}
	slices.Sort(want)
	if ids := openPollIDs(got); !slices.Equal(ids, want) {
		t.Fatalf("listed %v, want %v", ids, want)
	}
}

func TestACallerInNoGroupGetsAnEmptyListNotNull(t *testing.T) {
	f := newPolls(t)
	f.poll(t, nil)
	got, err := f.pm.ListOpenPolls(context.Background(), "nobody")
	if err != nil {
		t.Fatalf("ListOpenPolls: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("list = %#v, want an empty non-nil list", got)
	}
	raw, err := json.Marshal(got)
	if err != nil || string(raw) != "[]" {
		t.Fatalf("json = %s, %v", raw, err)
	}
}

func TestAManagerGivenNoGroupsListsNothing(t *testing.T) {
	f := newPolls(t)
	f.poll(t, nil)
	f.pm.UseGroups(nil)
	got, err := f.pm.ListOpenPolls(context.Background(), "amina")
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("list = %#v, %v", got, err)
	}
}

func TestAnOpenPollListedToARespondentNamesNeitherWhoMadeItNorWhoPublishedIt(t *testing.T) {
	f := newPolls(t)
	p := f.poll(t, nil)
	if _, err := f.pm.PublishResult(context.Background(), Publication{PollID: p.ID, PublishedBy: "clerk", GroupIDs: []string{"ward-a"}}); err != nil {
		t.Fatalf("PublishResult: %v", err)
	}
	got, err := f.pm.ListOpenPolls(context.Background(), "amina")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if strings.Contains(text, "created_by") || strings.Contains(text, "published_by") || strings.Contains(text, "clerk") {
		t.Fatalf("respondent list names a member: %s", text)
	}
	for _, field := range []string{`"id"`, `"question"`, `"choices"`, `"answered":false`} {
		if !strings.Contains(text, field) {
			t.Fatalf("respondent list lacks %s: %s", field, text)
		}
	}
}

func TestListingOpenPollsNeedsACaller(t *testing.T) {
	f := newPolls(t)
	f.poll(t, nil)
	if _, err := f.pm.ListOpenPolls(context.Background(), ""); !errors.Is(err, ErrNoReader) {
		t.Fatalf("err = %v, want ErrNoReader", err)
	}
}
