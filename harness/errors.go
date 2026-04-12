package harness

import "errors"

var (
	ErrNotFound      = errors.New("harness: not found")
	ErrConflict      = errors.New("harness: version conflict")
	ErrArchived      = errors.New("harness: agent is archived")
	ErrUnauthorized  = errors.New("harness: unauthorized")
	ErrRunNotActive  = errors.New("harness: run is not active")
	ErrTokenExpired  = errors.New("harness: vault token expired")
	ErrVerifyFailed  = errors.New("harness: tool verification failed")
	ErrSkillNotFound = errors.New("harness: skill not found")
)
