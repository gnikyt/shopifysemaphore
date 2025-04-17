package shopifysemaphore

import (
	"context"
	"sync"
	"time"
)

var (
	DefaultAquireBuffer = 200 * time.Millisecond // Default aquire throttle duration.
	DefaultPauseBuffer  = 1 * time.Second        // Default pause buffer to append to pause duration calculation.
)

// Semaphore is responsible regulating when to pause and resume processing of Goroutines.
// Points remaining, point thresholds, and point refill rates are taken into
// consideration. If remaining points go below the threshold, a pause is initiated
// which will also calculate how long a pause should happen based on the refill rate.
// Once pause is completed, the processing will resume. A PauceFunc and ResumeFunc
// can optionally be passed in which will fire respectively when a pause happens
// and when a resume happens.
type Semaphore struct {
	*Balance // Point information and tracking.

	PauseFunc    func(int32, time.Duration) // Optional callback for when pause happens.
	ResumeFunc   func()                     // Optional callback for when resume happens.
	PauseBuffer  time.Duration              // Buffer of time to wait before attempting to re-aquire a spot.
	AquireBuffer time.Duration              // Buffer of time to extend the pause with.

	pausedAt time.Time     // When paused last happened.
	sema     chan struct{} // Semaphore for controlling the number of Goroutines running.

	mu     sync.Mutex // For handling paused flag control.
	paused bool       // Pause flag.
}

// NewSemaphore returns a pointer to Semaphore. It accepts a cap which represents the
// capacity of how many Goroutines can run at a time, it also accepts information
// about the point balance and lastly, optional parameters.
func NewSemaphore(cap int, b *Balance, opts ...func(*Semaphore)) *Semaphore {
	sem := &Semaphore{
		Balance: b,
		sema:    make(chan struct{}, cap),
	}
	for _, opt := range opts {
		opt(sem)
	}
	if sem.PauseFunc == nil {
		// Provide default PauseFunc.
		WithPauseFunc(func(_ int32, _ time.Duration) {})(sem)
	}
	if sem.ResumeFunc == nil {
		// Provide default ResumeFunc.
		WithResumeFunc(func() {})(sem)
	}
	if sem.AquireBuffer == 0 {
		WithAquireBuffer(DefaultAquireBuffer)(sem)
	}
	return sem
}

// Aquire will attempt to aquire a spot to run the Goroutine.
// It will continue in a loop until it does aquire also pausing
// if the pause flag has been enabled. Aquiring is throttled at
// the value of AquireBuffer.
func (sem *Semaphore) Aquire(ctx context.Context) (err error) {
	for aquired := false; !aquired; {
		for {
			if !sem.paused {
				// Not paused. Break loop.
				break
			}
		}

		// Attempt to aquire a spot, if not we will throttle the next loop.
		select {
		case <-ctx.Done():
			// Context cancelled. Break loop and return error.
			aquired = true
			err = ctx.Err()
		case sem.sema <- struct{}{}:
			// Spot aquired. Break loop.
			aquired = true
		default:
			// Can not yet aquire a spot. Throttle for a set duration.
			time.Sleep(sem.AquireBuffer)
		}
	}
	return
}

// Release will release a spot for another Goroutine to take.
// It accepts a current value of remaining point balance, to which the
// remaining point balance will only be updated if the count is greater than -1.
// If the remaining points is below the set threshold, a pause will be
// initiated and a duration of this pause will be calculated based
// upon several factors surrouding the point information such as limit,
// threshold, and the refull rate.
func (sem *Semaphore) Release(pts int32) {
	defer sem.mu.Unlock()
	sem.mu.Lock()

	sem.Update(pts)
	if sem.AtThreshold() {
		// Calculate the duration required to refill and that duration time
		// has passed before we call for a pause.
		ra := sem.RefillDuration() + sem.PauseBuffer
		if sem.pausedAt.Add(ra).Before(time.Now()) {
			sem.paused = true
			sem.pausedAt = time.Now()
			go sem.PauseFunc(pts, ra)

			// Unflag as paused after the determined duration and run the ResumeFunc.
			go func() {
				time.Sleep(ra)
				sem.paused = false
				sem.ResumeFunc()
			}()
		}
	}

	// Perform the actual release.
	<-sem.sema
}

// withPauseFunc is a functional option for Semaphore to call when
// a pause happens. The point balance remaining and the duration of
// the pause will passed into the function.
func WithPauseFunc(fn func(int32, time.Duration)) func(*Semaphore) {
	return func(sem *Semaphore) {
		sem.PauseFunc = fn
	}
}

// withResumeFunc is a functional option for Semaphore to call when
// resume from a pause happens.
func WithResumeFunc(fn func()) func(*Semaphore) {
	return func(sem *Semaphore) {
		sem.ResumeFunc = fn
	}
}

// WithAquireBuffer is a functional option for Semaphore which
// will set the throttle duration for attempting to re-aquire a spot.
func WithAquireBuffer(dur time.Duration) func(*Semaphore) {
	return func(sem *Semaphore) {
		sem.AquireBuffer = dur
	}
}

// WithPauseBuffer is a functional option for Semaphore which
// will set an additional duration to append to the pause duration.
func WithPauseBuffer(dur time.Duration) func(*Semaphore) {
	return func(sem *Semaphore) {
		sem.PauseBuffer = dur
	}
}
