package shopifysemaphore

// TODO: Test methods can be 100% improved, but unable to at the moment.

import (
	"context"
	"sync"
	"testing"
	"time"
)

func newSemaphore(cap int, opts ...func(*Semaphore)) *Semaphore {
	return NewSemaphore(cap, NewBalance(900, 1000, 100), opts...)
}

func TestAquireOfOneCapacity(t *testing.T) {
	var err error
	var cnt int
	var wg sync.WaitGroup
	wg.Add(2)

	dur := 500 * time.Millisecond
	ctx, cn := context.WithTimeout(context.Background(), dur)
	defer cn()

	sema := newSemaphore(1)
	for i := 0; i < 2; i += 1 {
		go func() {
			err = sema.Aquire(ctx)
			if err != nil {
				// Timeout happened.
				wg.Done()
				return
			}

			// Increase aquired count.
			cnt += 1
			// Fake work.
			time.Sleep(dur / 10)

			sema.Release(1000)
			wg.Done()
		}()
	}
	wg.Wait()

	if cnt != 2 {
		t.Errorf("aquire count = %d; want 2", cnt)
	}
	if err != nil {
		t.Errorf("err = %v; want nil", err)
	}
}

func TestAquireOfOneCapacityWithLongWork(t *testing.T) {
	var err error
	var cnt int
	var wg sync.WaitGroup
	wg.Add(2)

	dur := 200 * time.Millisecond
	ctx, cn := context.WithTimeout(context.Background(), dur)
	defer cn()

	sema := newSemaphore(1)
	for i := 0; i < 2; i += 1 {
		go func() {
			err = sema.Aquire(ctx)
			if err != nil {
				// Timeout happened.
				wg.Done()
				return
			}

			// Increase aquired count.
			cnt += 1
			// Fake long work.
			time.Sleep(dur * 4)

			sema.Release(1000)
			wg.Done()
		}()
	}
	wg.Wait()

	if cnt != 1 {
		t.Errorf("aquire count = %d; want 1", cnt)
	}
	if err == nil {
		t.Errorf("err = %v; want error", err)
	}
}

func TestReleaseCausesPauseAndResume(t *testing.T) {
	var cnt int
	var err error
	var res bool
	var wg sync.WaitGroup
	wg.Add(2)

	sema := newSemaphore(1, WithPauseFunc(func(i int32, d time.Duration) {
		if i != 900 || d.String() != "1s" {
			t.Errorf("PauseFunc(%d, %s); want PauseFunc(900, 1s)", i, d)
		}
	}), WithResumeFunc(func() {
		res = true
	}))

	for i := 0; i < 2; i += 1 {
		go func() {
			err = sema.Aquire(context.Background())
			if err != nil {
				wg.Done()
				return
			}

			// Increase aquired count.
			cnt += 1
			// Fake work.
			time.Sleep(50 * time.Millisecond)

			sema.Release(900)
			wg.Done()
		}()
	}
	wg.Wait()

	if cnt != 2 {
		t.Errorf("aquire count = %d; want 2", cnt)
	}
	if err != nil {
		t.Errorf("err = %v; want nil", err)
	}
	if res == false {
		t.Errorf("res = %v; want true", res)
	}
}
