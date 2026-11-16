package voting

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func windowedPoll(opens, closes string) Poll {
	return Poll{Question: "Which street?", Mode: "one", PerGroup: "once", GroupIDs: []string{"ward-a"}, CreatedBy: "clerk", OpensAt: opens, ClosesAt: closes, Choices: []Choice{{Label: "Mill Street"}}}
}

func TestAPollWindowEndThatIsNotAnRFC3339InstantIsRefusedNamingTheField(t *testing.T) {
	f := newPolls(t)
	notInstants := map[string]string{
		"february 30":          "2026-02-30T00:00:00Z",
		"hour 24":              "2026-11-08T24:00:00Z",
		"leap second":          "2016-12-31T23:59:60Z",
		"date only":            "2026-11-08",
		"no zone":              "2026-11-08T00:00:00",
		"lower case separator": "2026-11-08t00:00:00z",
		"leading space":        " 2026-11-08T00:00:00Z",
		"trailing space":       "2026-11-08T00:00:00Z ",
		"empty":                "",
	}
	for name, bad := range notInstants {
		t.Run("opens_at "+name, func(t *testing.T) {
			_, err := f.pm.CreatePoll(context.Background(), windowedPoll(bad, "2026-11-09T00:00:00Z"))
			if !errors.Is(err, ErrInvalidPoll) || !strings.Contains(err.Error(), "opens_at") {
				t.Fatalf("opens_at %q: %v", bad, err)
			}
		})
		t.Run("closes_at "+name, func(t *testing.T) {
			_, err := f.pm.CreatePoll(context.Background(), windowedPoll("2026-11-01T00:00:00Z", bad))
			if !errors.Is(err, ErrInvalidPoll) || !strings.Contains(err.Error(), "closes_at") {
				t.Fatalf("closes_at %q: %v", bad, err)
			}
		})
	}
}

func TestAPollWindowIsComparedByInstantNotByText(t *testing.T) {
	f := newPolls(t)
	refused := map[string][2]string{
		"closes in an earlier offset than it opens": {"2026-11-08T00:00:00Z", "2026-11-08T12:00:00+14:00"},
		"same instant in two offsets":               {"2026-11-08T12:00:00Z", "2026-11-08T14:00:00+02:00"},
	}
	for name, w := range refused {
		t.Run("refuses "+name, func(t *testing.T) {
			_, err := f.pm.CreatePoll(context.Background(), windowedPoll(w[0], w[1]))
			if !errors.Is(err, ErrInvalidPoll) || !strings.Contains(err.Error(), "closes_at falls after opens_at") {
				t.Fatalf("window %v: %v", w, err)
			}
		})
	}
	t.Run("accepts a ten hour window whose text sorts backwards", func(t *testing.T) {
		if _, err := f.pm.CreatePoll(context.Background(), windowedPoll("2026-11-08T05:00:00+14:00", "2026-11-08T00:00:00Z")); err != nil {
			t.Fatalf("real window refused: %v", err)
		}
	})
}

func TestAPollWindowIsStoredAsUTCInstants(t *testing.T) {
	f := newPolls(t)
	p, err := f.pm.CreatePoll(context.Background(), windowedPoll("2026-11-08T10:00:00+02:00", "2026-11-08T20:30:15.5+02:00"))
	if err != nil {
		t.Fatal(err)
	}
	if p.OpensAt != "2026-11-08T08:00:00Z" || p.ClosesAt != "2026-11-08T18:30:15.5Z" {
		t.Fatalf("returned %q to %q", p.OpensAt, p.ClosesAt)
	}
	got, err := f.pm.GetPoll(context.Background(), p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.OpensAt != p.OpensAt || got.ClosesAt != p.ClosesAt {
		t.Fatalf("stored %q to %q", got.OpensAt, got.ClosesAt)
	}
}

func TestStoredPollsKeepTheWindowTheyWereWrittenWithAndReadAsBefore(t *testing.T) {
	f := newPolls(t)
	ctx := context.Background()
	stored := func(opens, closes string) Poll {
		p := f.poll(t, nil)
		p.ID, p.Choices = newID(), nil
		p.OpensAt, p.ClosesAt = opens, closes
		if err := f.pm.st.Insert(ctx, objectPoll, p); err != nil {
			t.Fatal(err)
		}
		return p
	}
	text := stored("2026-11-08", "2026-11-09 18:00")
	offsets := stored(f.now.Add(-time.Hour).In(time.FixedZone("west", -5*3600)).Format(time.RFC3339), f.now.Add(time.Hour).In(time.FixedZone("east", 14*3600)).Format(time.RFC3339))

	for _, p := range []Poll{text, offsets} {
		got, err := f.pm.GetPoll(ctx, p.ID)
		if err != nil || got.OpensAt != p.OpensAt || got.ClosesAt != p.ClosesAt {
			t.Fatalf("stored window changed: %+v %v", got, err)
		}
	}
	if err := f.pm.OpenToReader(ctx, text.ID, "amina"); !errors.Is(err, ErrPollNotFound) {
		t.Fatalf("text window poll read by a resident: %v", err)
	}
	if _, err := f.pm.AnswerPoll(ctx, Answer{PollID: text.ID, AnsweredBy: "amina"}); !errors.Is(err, ErrOutsideWindow) {
		t.Fatalf("text window poll took an answer: %v", err)
	}
	if err := f.pm.OpenToReader(ctx, offsets.ID, "amina"); err != nil {
		t.Fatalf("offset window poll closed to a resident: %v", err)
	}
	if !insideWindow(offsets, f.pm.now()) {
		t.Fatal("offset window poll is not open by instant")
	}
}
