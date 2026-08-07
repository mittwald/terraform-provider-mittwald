package apiutils_test

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/mittwald/terraform-provider-mittwald/internal/apiutils"
)

var testOpts = apiutils.PollOpts{
	InitialDelay:  time.Millisecond,
	MaxDelay:      10 * time.Millisecond,
	BackoffFactor: 1.1,
}

func TestPollReturnsResultOnSuccess(t *testing.T) {
	ctx := context.Background()

	res, err := apiutils.Poll(ctx, testOpts, func(_ context.Context, p int) (int, error) {
		return p * 2, nil
	}, 21)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res != 42 {
		t.Fatalf("expected 42, got %d", res)
	}
}

func TestPollRetriesUntilSuccess(t *testing.T) {
	ctx := context.Background()

	calls := 0
	res, err := apiutils.Poll(ctx, testOpts, func(_ context.Context, _ struct{}) (string, error) {
		calls++
		if calls < 3 {
			return "", apiutils.ErrPollShouldRetry
		}
		return "done", nil
	}, struct{}{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res != "done" {
		t.Fatalf("expected \"done\", got %q", res)
	}

	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestPollReturnsNonRetryableError(t *testing.T) {
	ctx := context.Background()
	sentinel := errors.New("boom")

	_, err := apiutils.Poll(ctx, testOpts, func(_ context.Context, _ struct{}) (string, error) {
		return "", sentinel
	}, struct{}{})

	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}

// TestPollDoesNotPanicWhenContextIsCancelledDuringCall reproduces the race
// described in #448: the polling function is still running when the context is
// cancelled, and returns a successful result afterwards.
func TestPollDoesNotPanicWhenContextIsCancelledDuringCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	entered := make(chan struct{})
	release := make(chan struct{})

	done := make(chan error, 1)
	go func() {
		_, err := apiutils.Poll(ctx, testOpts, func(_ context.Context, _ struct{}) (string, error) {
			close(entered)
			<-release
			return "late result", nil
		}, struct{}{})
		done <- err
	}()

	<-entered
	cancel()

	// Give Poll a chance to observe the cancellation before the polling
	// function returns its (now unwanted) result.
	time.Sleep(50 * time.Millisecond)
	close(release)

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Poll did not return after the context was cancelled")
	}
}

// TestPollDoesNotLeakGoroutines covers the secondary issue in #448: a Poll call
// that returns because its context expired used to leave a goroutine parked on
// the ticker channel forever.
func TestPollDoesNotLeakGoroutines(t *testing.T) {
	before := runtime.NumGoroutine()

	for range 20 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		_, err := apiutils.Poll(ctx, testOpts, func(_ context.Context, _ struct{}) (string, error) {
			return "", apiutils.ErrPollShouldRetry
		}, struct{}{})
		cancel()

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected context.DeadlineExceeded, got %v", err)
		}
	}

	// Allow any goroutines that are legitimately winding down to exit.
	for range 50 {
		if runtime.NumGoroutine() <= before+2 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("goroutine count grew from %d to %d", before, runtime.NumGoroutine())
}
