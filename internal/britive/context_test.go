package britive

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestContextWithDefaultDeadline_NoParentDeadline(t *testing.T) {
	ctx, cancel := contextWithDefaultDeadline(context.Background(), 100*time.Millisecond)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected derived context to have a deadline, got none")
	}
	if until := time.Until(deadline); until > 100*time.Millisecond {
		t.Errorf("deadline too far in the future: %v", until)
	}
}

func TestContextWithDefaultDeadline_ParentHasDeadline(t *testing.T) {
	parent, parentCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer parentCancel()

	ctx, cancel := contextWithDefaultDeadline(parent, 1*time.Hour)
	defer cancel()

	// The returned context should inherit the parent's deadline, not the default.
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected context to have a deadline, got none")
	}
	if until := time.Until(deadline); until > 100*time.Millisecond {
		t.Errorf("expected parent's 50ms deadline to be preserved, got %v remaining", until)
	}
}

func TestSleepCtx_Timeout(t *testing.T) {
	start := time.Now()
	if err := sleepCtx(context.Background(), 10*time.Millisecond); err != nil {
		t.Errorf("sleepCtx() with no cancellation returned error: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 10*time.Millisecond {
		t.Errorf("sleepCtx returned too early: %v", elapsed)
	}
}

func TestSleepCtx_Canceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	start := time.Now()
	err := sleepCtx(ctx, 1*time.Hour)
	if err == nil {
		t.Fatal("sleepCtx() with canceled context returned nil, want error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("sleepCtx() error = %v, want context.Canceled", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Errorf("sleepCtx did not return promptly: %v", elapsed)
	}
}
