package voting

import "errors"

var (
	ErrPollNotFound  = errors.New("poll not found")
	ErrInvalidPoll   = errors.New("invalid poll")
	ErrInvalidChoice = errors.New("invalid choice")
	ErrNoCreator     = errors.New("a poll needs a caller to stand behind it")
	ErrPollNotDraft  = errors.New("a poll's choices change only while it is a draft")
)
