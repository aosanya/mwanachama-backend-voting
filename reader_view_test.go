package voting

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAPollIsOpenToAReaderOnlyOnceItHasOpenedAndOneOfItsGroupsIsTheirs(t *testing.T) {
	f := newPolls(t)
	ctx := context.Background()
	p := f.poll(t, nil)
	future := f.poll(t, func(p *Poll) {
		p.OpensAt = f.now.Add(time.Hour).Format(time.RFC3339)
		p.ClosesAt = f.now.Add(2 * time.Hour).Format(time.RFC3339)
	})
	closed := f.poll(t, func(p *Poll) {
		p.OpensAt = f.now.Add(-2 * time.Hour).Format(time.RFC3339)
		p.ClosesAt = f.now.Add(-time.Hour).Format(time.RFC3339)
	})
	cases := []struct {
		name, poll, reader string
		want               error
	}{
		{"a member of the poll's group", p.ID, "amina", nil},
		{"a member of several groups", p.ID, "otieno", nil},
		{"a closed poll stays readable to its groups", closed.ID, "amina", nil},
		{"a poll not yet open", future.ID, "amina", ErrPollNotFound},
		{"a poll aimed at other groups", p.ID, "stranger", ErrPollNotFound},
		{"a reader in no group", p.ID, "nobody", ErrPollNotFound},
		{"a poll that does not exist", "missing", "amina", ErrPollNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := f.pm.OpenToReader(ctx, c.poll, c.reader); !errors.Is(err, c.want) {
				t.Fatalf("OpenToReader = %v, want %v", err, c.want)
			}
		})
	}
}

func TestAPollReadAsARespondentNamesNeitherWhoMadeItNorWhoPublishedIt(t *testing.T) {
	f := newPolls(t)
	p := f.poll(t, nil)
	if _, err := f.pm.PublishResult(context.Background(), Publication{PollID: p.ID, PublishedBy: "clerk", GroupIDs: []string{"ward-a"}}); err != nil {
		t.Fatalf("PublishResult: %v", err)
	}
	shown, err := f.pm.ShowPoll(context.Background(), p.ID)
	if err != nil {
		t.Fatal(err)
	}
	staff, err := json.Marshal(shown)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(staff), `"created_by":"clerk"`) || !strings.Contains(string(staff), `"published_by":"clerk"`) {
		t.Fatalf("staff read = %s", staff)
	}
	poll, ok := shown.(Poll)
	if !ok {
		t.Fatalf("ShowPoll returned %T", shown)
	}
	resident, err := json.Marshal(poll.RespondentView())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(resident), "created_by") || strings.Contains(string(resident), "published_by") || strings.Contains(string(resident), "clerk") {
		t.Fatalf("respondent read names a member: %s", resident)
	}
	if !strings.Contains(string(resident), p.Question) {
		t.Fatalf("respondent read lost the question: %s", resident)
	}
}

func TestShowingAPollThatDoesNotExistAnswersNotFound(t *testing.T) {
	f := newPolls(t)
	if _, err := f.pm.ShowPoll(context.Background(), "missing"); !errors.Is(err, ErrPollNotFound) {
		t.Fatalf("ShowPoll = %v, want ErrPollNotFound", err)
	}
}
