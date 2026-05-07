package auth

import (
	"fmt"
	"sync"
	"time"

	"pos-backend/internal/pos/domain"
)

const (
	defaultPINAttemptsWindow = 10 * time.Minute
	defaultPINLockoutWindow  = 15 * time.Minute
	defaultPINAttemptsLimit  = 5
)

type pinRateLimiter struct {
	mu             sync.Mutex
	attempts       map[string]pinAttemptState
	maxAttempts    int
	attemptWindow  time.Duration
	lockoutWindow  time.Duration
}

type pinAttemptState struct {
	windowStart time.Time
	attempts    int
	lockedUntil time.Time
}

func newPINRateLimiter(maxAttempts int, attemptWindow, lockoutWindow time.Duration) *pinRateLimiter {
	return &pinRateLimiter{
		attempts:      make(map[string]pinAttemptState),
		maxAttempts:   maxAttempts,
		attemptWindow: attemptWindow,
		lockoutWindow: lockoutWindow,
	}
}

func (l *pinRateLimiter) Allow(key string, now time.Time) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	state, ok := l.attempts[key]
	if !ok {
		return nil
	}
	if !state.lockedUntil.IsZero() && now.Before(state.lockedUntil) {
		return fmt.Errorf("%w: retry later", domain.ErrTooManyRequests)
	}
	if state.windowStart.IsZero() || now.Sub(state.windowStart) >= l.attemptWindow {
		delete(l.attempts, key)
	}
	return nil
}

func (l *pinRateLimiter) RecordFailure(key string, now time.Time) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	state := l.attempts[key]
	if state.windowStart.IsZero() || now.Sub(state.windowStart) >= l.attemptWindow {
		state.windowStart = now
		state.attempts = 0
		state.lockedUntil = time.Time{}
	}
	state.attempts++
	if state.attempts >= l.maxAttempts {
		state.lockedUntil = now.Add(l.lockoutWindow)
		l.attempts[key] = state
		return fmt.Errorf("%w: retry later", domain.ErrTooManyRequests)
	}
	l.attempts[key] = state
	return nil
}

func (l *pinRateLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}
