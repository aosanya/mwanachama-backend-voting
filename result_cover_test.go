package voting

import (
	"context"
	"errors"
	"testing"
)

func publishedPoll(t *testing.T, m *pollManager, groups ...string) Poll {
	t.Helper()
	p := aPollWith(t, m, nil)
	out, err := m.PublishResult(context.Background(), Publication{PollID: p.ID, PublishedBy: "holder-1", GroupIDs: groups})
	if err != nil {
		t.Fatalf("PublishResult: %v", err)
	}
	return out
}

func TestPublishingNeedsACallerAndAtLeastOneGroup(t *testing.T) {
	m := answeringManager(t, insideTheWindow)
	p := aPollWith(t, m, nil)
	ctx := context.Background()
	if _, err := m.PublishResult(ctx, Publication{PollID: p.ID, GroupIDs: []string{"g-1"}}); !errors.Is(err, ErrNoPublisher) {
		t.Fatalf("no publisher err = %v, want ErrNoPublisher", err)
	}
	if _, err := m.PublishResult(ctx, Publication{PollID: p.ID, PublishedBy: "holder-1"}); !errors.Is(err, ErrInvalidPublication) {
		t.Fatalf("no groups err = %v, want ErrInvalidPublication", err)
	}
	got, err := m.GetPoll(ctx, p.ID)
	if err != nil || got.PublishedAt != "" || got.PublishedBy != "" {
		t.Fatalf("poll after refusals = %+v, %v", got, err)
	}
}

func TestPublishingAMissingPollIsNotFound(t *testing.T) {
	m := answeringManager(t, insideTheWindow)
	_, err := m.PublishResult(context.Background(), Publication{PollID: "nope", PublishedBy: "holder-1", GroupIDs: []string{"g-1"}})
	if !errors.Is(err, ErrPollNotFound) {
		t.Fatalf("err = %v, want ErrPollNotFound", err)
	}
}

func TestPublishingRecordsWhenByWhomAndToWhichGroups(t *testing.T) {
	m := answeringManager(t, insideTheWindow)
	p := publishedPoll(t, m, "g-1", "g-2")
	if p.PublishedAt != insideTheWindow || p.PublishedBy != "holder-1" {
		t.Fatalf("published = %+v", p)
	}
	if len(p.PublishedGroupIDs) != 2 || p.PublishedGroupIDs[0] != "g-1" || p.PublishedGroupIDs[1] != "g-2" {
		t.Fatalf("published groups = %v", p.PublishedGroupIDs)
	}
	if p.Question == "" || len(p.GroupIDs) != 2 || p.State != "draft" {
		t.Fatalf("publishing disturbed the poll: %+v", p)
	}
}

func TestPublishingAgainReplacesThePublisherAndGroups(t *testing.T) {
	m := answeringManager(t, insideTheWindow)
	p := publishedPoll(t, m, "g-1")
	out, err := m.PublishResult(context.Background(), Publication{PollID: p.ID, PublishedBy: "holder-2", GroupIDs: []string{"g-2"}})
	if err != nil {
		t.Fatalf("second publish: %v", err)
	}
	if out.PublishedBy != "holder-2" || len(out.PublishedGroupIDs) != 1 || out.PublishedGroupIDs[0] != "g-2" {
		t.Fatalf("republished = %+v", out)
	}
}

func TestAPublishedCountIsReadOnlyByAMemberOfAPublishedGroup(t *testing.T) {
	m := answeringManager(t, insideTheWindow)
	p := aPollWith(t, m, nil)
	ctx := context.Background()
	if _, err := answerAs(m, ctx, Answer{PollID: p.ID, AnsweredBy: "a-1", Votes: votesOn(p, 1)}, []string{"g-1"}); err != nil {
		t.Fatalf("answer: %v", err)
	}
	if _, err := m.PublishResult(ctx, Publication{PollID: p.ID, PublishedBy: "holder-1", GroupIDs: []string{"g-2"}}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	m.UseGroups(func(context.Context, string) ([]string, error) { return []string{"g-2"}, nil })
	c, err := m.ReadPublishedCount(ctx, p.ID, "reader")
	if err != nil {
		t.Fatalf("ReadPublishedCount: %v", err)
	}
	if c.Answered != 1 || c.Choices[0].Votes != 1 || c.PollID != p.ID {
		t.Fatalf("count = %+v", c)
	}
	m.UseGroups(func(context.Context, string) ([]string, error) { return []string{"g-1"}, nil })
	if _, err := m.ReadPublishedCount(ctx, p.ID, "reader"); !errors.Is(err, ErrPollNotFound) {
		t.Fatalf("outside group err = %v, want ErrPollNotFound", err)
	}
}

func TestAnUnpublishedCountAnswersLikeAMissingPoll(t *testing.T) {
	m := answeringManager(t, insideTheWindow)
	p := aPollWith(t, m, nil)
	m.UseGroups(func(context.Context, string) ([]string, error) { return []string{"g-1"}, nil })
	_, unpublished := m.ReadPublishedCount(context.Background(), p.ID, "reader")
	_, missing := m.ReadPublishedCount(context.Background(), "nope", "reader")
	if !errors.Is(unpublished, ErrPollNotFound) || unpublished.Error() != missing.Error() {
		t.Fatalf("unpublished = %v, missing = %v", unpublished, missing)
	}
}

func TestReadingAPublishedCountNeedsACaller(t *testing.T) {
	m := answeringManager(t, insideTheWindow)
	p := publishedPoll(t, m, "g-1")
	if _, err := m.ReadPublishedCount(context.Background(), p.ID, ""); !errors.Is(err, ErrNoReader) {
		t.Fatalf("err = %v, want ErrNoReader", err)
	}
}

func TestAManagerGivenNoGroupsShowsAPublishedCountToNobody(t *testing.T) {
	m := answeringManager(t, insideTheWindow)
	p := publishedPoll(t, m, "g-1")
	if _, err := m.ReadPublishedCount(context.Background(), p.ID, "reader"); !errors.Is(err, ErrPollNotFound) {
		t.Fatalf("err = %v, want ErrPollNotFound", err)
	}
}

func TestAGroupLookupFailureIsReportedNotReadAsNoGroups(t *testing.T) {
	m := answeringManager(t, insideTheWindow)
	p := publishedPoll(t, m, "g-1")
	boom := errors.New("directory down")
	m.UseGroups(func(context.Context, string) ([]string, error) { return nil, boom })
	_, err := m.ReadPublishedCount(context.Background(), p.ID, "reader")
	if !errors.Is(err, boom) || errors.Is(err, ErrPollNotFound) {
		t.Fatalf("err = %v, want the lookup failure", err)
	}
}
