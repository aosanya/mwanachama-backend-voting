package voting

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestAPollWithAControlByteInItsQuestionOrAChoiceIsRefused(t *testing.T) {
	f := newPolls(t)
	opens, closes := f.window()
	cases := map[string]struct {
		edit func(*Poll)
		want error
	}{
		"question": {func(p *Poll) { p.Question = "Which street\x00 first?" }, ErrInvalidPoll},
		"choice":   {func(p *Poll) { p.Choices = []Choice{{Label: "Mill\x07 Street"}, {Label: "Orchard Lane"}} }, ErrInvalidChoice},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			in := Poll{Question: "Which street first?", Mode: "one", PerGroup: "once", GroupIDs: []string{"ward-a"}, CreatedBy: "clerk", OpensAt: opens, ClosesAt: closes, Choices: []Choice{{Label: "Mill Street"}, {Label: "Orchard Lane"}}}
			c.edit(&in)
			if _, err := f.pm.CreatePoll(context.Background(), in); !errors.Is(err, c.want) || !strings.Contains(err.Error(), "not plain text") {
				t.Fatalf("CreatePoll = %v, want %v for control characters", err, c.want)
			}
		})
	}
}

func TestLineBreaksAndTabsStayInAPollQuestion(t *testing.T) {
	f := newPolls(t)
	p := f.poll(t, func(p *Poll) { p.Question = "Which street\nis resurfaced\tfirst?" })
	if p.Question != "Which street\nis resurfaced\tfirst?" {
		t.Fatalf("question = %q", p.Question)
	}
}
