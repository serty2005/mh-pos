package auth

import (
	"errors"
	"testing"
	"time"

	"pos-backend/internal/pos/domain"
)

func TestPINRateLimiterLocksAfterConfiguredAttempts(t *testing.T) {
	limiter := newPINRateLimiter(3, 10*time.Minute, 15*time.Minute)
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	key := "node-1:client-1"

	if err := limiter.Allow(key, now); err != nil {
		t.Fatalf("expected first attempt allowed, got %v", err)
	}
	if err := limiter.RecordFailure(key, now); err != nil {
		t.Fatalf("unexpected lock on attempt 1: %v", err)
	}
	if err := limiter.RecordFailure(key, now.Add(time.Minute)); err != nil {
		t.Fatalf("unexpected lock on attempt 2: %v", err)
	}
	if err := limiter.RecordFailure(key, now.Add(2*time.Minute)); !errors.Is(err, domain.ErrTooManyRequests) {
		t.Fatalf("expected lock on attempt 3, got %v", err)
	}
	if err := limiter.Allow(key, now.Add(3*time.Minute)); !errors.Is(err, domain.ErrTooManyRequests) {
		t.Fatalf("expected lockout to be active, got %v", err)
	}
}

func TestPINRateLimiterResetsAfterLockoutAndSuccess(t *testing.T) {
	limiter := newPINRateLimiter(2, 10*time.Minute, 15*time.Minute)
	base := time.Date(2026, 5, 5, 13, 0, 0, 0, time.UTC)
	key := "node-1:client-1"

	_ = limiter.RecordFailure(key, base)
	_ = limiter.RecordFailure(key, base.Add(time.Minute))
	if err := limiter.Allow(key, base.Add(2*time.Minute)); !errors.Is(err, domain.ErrTooManyRequests) {
		t.Fatalf("expected active lockout, got %v", err)
	}
	if err := limiter.Allow(key, base.Add(17*time.Minute)); err != nil {
		t.Fatalf("expected lockout window to expire, got %v", err)
	}

	limiter.Reset(key)
	if err := limiter.Allow(key, base.Add(18*time.Minute)); err != nil {
		t.Fatalf("expected allow after reset, got %v", err)
	}
}
