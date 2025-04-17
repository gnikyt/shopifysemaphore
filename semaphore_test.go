package shopifysemaphore

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func newSemaphore(cap int, opts ...func(*Semaphore)) *Semaphore {
	return NewSemaphore(cap, NewBalance(900, 1000, 100), opts...)
}

// TestAquire should run N Goroutines. Allowing them
// to aquire and release their spot. We are expecting no error to happen
// and for the count (cnt) to match the number of Goroutines N.
func TestAquire(t *testing.T) {
	var err error
	var wg sync.WaitGroup
	var cnt int // Count of Gorotunes which ran.

	cap := 1 // Capacity.
	n := 2   // Number of Goroutines to spin up.

	ctx := context.Background()
	sema := newSemaphore(cap)
	for i := 0; i < n; i += 1 {
		wg.Add(1)
		go func() {
			err = sema.Aquire(ctx)
			if err != nil {
				wg.Done()
				return
			}

			cnt += 1

			wg.Done()
			sema.Release(1000)
		}()
	}
	wg.Wait()

	if cnt != n {
		t.Errorf("cnt = %d; want %d", cnt, n)
	}
	if err != nil {
		t.Errorf("Aquire(%q) = %v; want nil", ctx, err)
	}
}

// TestReleaseCausesPauseAndResume should create a situation where
// Release triggers a pause to happen due to hitting a threshold.
func TestReleaseCausesPauseAndResume(t *testing.T) {
	var err error
	var wg sync.WaitGroup
	var cnt int // Count of Gorotunes which ran.

	cap := 1                 // Capacity.
	n := 2                   // Number of Goroutines to spin up.
	res := false             // If resume happened.
	exdur := 1 * time.Second // Expected duration of pause.
	var expts int32 = 900    // Expected points at pause.

	ctx := context.Background()
	sema := newSemaphore(cap, WithPauseFunc(func(pts int32, dur time.Duration) {
		if pts != expts || dur != exdur {
			t.Errorf("PauseFunc(%d, %q); want PauseFunc(%d, %q)", pts, dur, expts, exdur)
		}
	}), WithResumeFunc(func() {
		res = true
	}))

	for i := 0; i < n; i += 1 {
		wg.Add(1)
		go func() {
			err = sema.Aquire(ctx)
			if err != nil {
				wg.Done()
				return
			}

			cnt += 1

			wg.Done()
			sema.Release(expts)
		}()
	}
	wg.Wait()

	if cnt != 2 {
		t.Errorf("cnt = %d; want %d", cnt, n)
	}
	if err != nil {
		t.Errorf("err = %v; want nil", err)
	}
	if res == false {
		t.Errorf("res = %v; want true", res)
	}
}

// TestAquireCtxErr should detect a context error on attempting
// to aquire a spot.
func TestAquireCtxErr(t *testing.T) {
	var err error
	var wg sync.WaitGroup

	cap := 1                      // Capacity.
	n := 2                        // Number of Goroutines to spin up.
	ctxo := 50 * time.Millisecond // Context timeout duration.

	ctx, cancel := context.WithTimeout(context.Background(), ctxo)
	defer cancel()

	sema := newSemaphore(cap)
	for i := 0; i < n; i += 1 {
		wg.Add(1)
		go func() {
			err = sema.Aquire(ctx)
			if err != nil {
				wg.Done()
				return
			}

			// Make work "long", longer than the context timeout.
			time.Sleep(ctxo * 3)

			wg.Done()
			sema.Release(1000)
		}()
	}
	wg.Wait()

	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v; want %v", err, context.DeadlineExceeded)
	}
}
