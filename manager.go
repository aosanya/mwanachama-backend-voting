package voting

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-voting/models"
)

type (
	Poll          = models.Poll
	Choice        = models.Choice
	Participation = models.Participation
	Ballot        = models.Ballot
)

type PollManager interface {
	CreatePoll(ctx context.Context, in Poll) (Poll, error)
	GetPoll(ctx context.Context, pollID string) (Poll, error)

	AddChoice(ctx context.Context, in Choice) (Choice, error)
	ListChoices(ctx context.Context, pollID string) ([]Choice, error)
}

type pollManager struct {
	db  *gorm.DB
	st  *store
	now func() string
}

func NewPollManager(db *gorm.DB, s *spec.Spec) (PollManager, error) {
	if db == nil {
		return nil, fmt.Errorf("NewPollManager: db must not be nil")
	}
	st, err := newStore(db, s, map[string]any{
		objectPoll:          models.Poll{},
		objectChoice:        models.Choice{},
		objectParticipation: models.Participation{},
		objectBallot:        models.Ballot{},
	})
	if err != nil {
		return nil, fmt.Errorf("NewPollManager: %w", err)
	}
	return &pollManager{db: db, st: st, now: nowStamp}, nil
}

func nowStamp() string { return time.Now().UTC().Format(specstore.TimeLayout) }
