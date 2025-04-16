# Shopify Semaphore

A service to assist with running Goroutines against limits of Shopify's GraphQL API. This service will "pause" running Goroutines if the configured API thresholds for point balance has been reached, pausing for a calculated duration, allowing for the point balance to refill completely before resuming.

## Installation

`go get github.com/gnikyt/shopifysemaphore`

No external dependencies for this package.

## Usage

Create a new Semaphore instance by supplying the capacity of the number of Goroutines you wish to run concurrently and information about the GraphQL point balance.

`Aquire(ctx context.Context)` accepts a context which will return an error, if one has happened such as a context timeout.

`Release(pts int32)` accepts an integer representing the remaining point balance returned by Shopify's GraphQL API response.

Example usage:

```go
package main

import ssem "github.com/gniktr/shopifysemaphore"

func work(id int, wg *sync.WaitGroup, ctx context.Context, sem *shopifysemaphore.Semaphore) {
  err := sem.Aquire(ctx)
  if err != nil {
    // Context timeout.
    wg.Done()
    return
  }

  points, err := graphQLCall() // Return remaining points from call.
  if err != nil {
    // Handle error.
  }
  fmt.Printf("remaining: %d points", points)
  sem.Release(points)
}

func main() {
  fmt.Println("started!")
  done := make(chan bool)
  ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Minute)

  // Semaphore with a concurrent capacity of 10.
  // Including a point balance setup with a threshold to pause at 200 points,
  // a maximum of 2000 points available, and a refill rate of 100 points per second.
  sem := ssem.NewSemaphore(
    10,
    ssem.NewBalance(200, 2000, 100),
    ssem.WithPauseFunc(func (pts int32, dur time.Duration) {
      fmt.Printf("pausing for %s due to remaining points of %d...", dur, pts)
    }),
    ssem.WithResumeFunc(func () {
      fmt.Println("resuming...")
    })
  )

  // Run 100 Goroutines.
  var wg sync.WaitGroup
  for i := 0; i < 100; i += 1 {
    wg.Add(1)
    go work(i, &wg, ctx, sem)
  }

  // Wait for completion of Goroutines.
	go func() {
		wg.Wait()
		done <- true
	}()

  select {
    case <-ctx.Done():
      fmt.Println("timeout happened.")
    case <-done:
      fmt.Println("work finished.")
  }
  fmt.Println("completed.")
}
```

Example output:

```
started!
remaining: 1840 points
remaining: 1710 points
remaining: 1660 points
...
remaining: 280 points
remaining: 190 points
pausing for 18 seconds due to remaining points of 190...
resuming...
remaining: 1890 points
remaining: 1810 points
...
work finished.
completed.
```

## Testing

`go test -v ./...`

## Documentation

```
// go doc -all
package shopifysemaphore // import "github.com/gnikyt/shopify-semaphore"


VARIABLES

var (
	DefaultAquireBuffer = 200 * time.Millisecond
	DefaultPauseBuffer  = 1 * time.Second
)

FUNCTIONS

func WithAquireBuffer(dur time.Duration) func(*Semaphore)
    WithAquireBuffer is a functional option for Semaphore which will set the
    throttle duration for attempting to re-aquire a spot.

func WithPauseBuffer(dur time.Duration) func(*Semaphore)
    WithPauseBuffer is a functional option for Semaphore which will set an
    additional duration to append to the pause duration.

func WithPauseFunc(fn func(int32, time.Duration)) func(*Semaphore)
    withPauseFunc is a functional option for Semaphore to call when a pause
    happens. The point balance remaining and the duration of the pause will
    passed into the function.

func WithResumeFunc(fn func()) func(*Semaphore)
    withResumeFunc is a functional option for Semaphore to call when resume from
    a pause happens.


TYPES

type Balance struct {
	Remaining  atomic.Int32 // Point balance remaining.
	Threshold  int32        // Minimum point balance where we would consider handling with a "pause".
	Limit      int32        // Maximum points available.
	RefillRate int32        // Number of points refilled per second.
}
    Balance represents the information of point values and keeps track of items
    such as the remaining points, threshold, limit, and refill rate.

func NewBalance(thld int32, max int32, rr int32) *Balance
    NewBalance accepts a threshold (thld) point balance, a maximum (max) point
    balance, and the refill rate (rr). It will return a pointer to Balance.

func (b *Balance) AtThreshold() bool
    AtThreshold will return a boolean if we have reached or surpassed the set
    threshold of remaining points or not.

func (b *Balance) RefillDuration() time.Duration
    RefillDuration accounts for the remaining points, the limit, and the refill
    rate to determine how many seconds it would take to refill to remaining
    points back to full. It will return a duration which can be used to "pause"
    operations.

func (b *Balance) Update(points int32)
    Update accepts a new value of remaining points to store.

type Semaphore struct {
	*Balance // Point information and tracking.

	PauseFunc    func(int32, time.Duration) // Optional callback for when pause happens.
	ResumeFunc   func()                     // Optional callback for when resume happens.
	PauseBuffer  time.Duration              // Buffer of time to wait before attempting to re-aquire a spot.
	AquireBuffer time.Duration              // Buffer of time to extend the pause with.

	// Has unexported fields.
}
    Semaphore is responsible regulating when to pause and resume processing
    of Goroutines. Points remaining, point thresholds, and point refill rates
    are taken into consideration. If remaining points go below the threshold,
    a pause is initiated which will also calculate how long a pause should
    happen based on the refill rate. Once pause is completed, the processing
    will resume. A PauceFunc and ResumeFunc can optionally be passed in which
    will fire respectively when a pause happens and when a resume happens.

func NewSemaphore(cap int, b *Balance, opts ...func(*Semaphore)) *Semaphore
    NewSemaphore returns a pointer to Semaphore. It accepts a cap which
    represents the capacity of how many Goroutines can run at a time, it also
    accepts information about the point balance and lastly, optional parameters.

func (sem *Semaphore) Aquire(ctx context.Context) (err error)
    Aquire will attempt to aquire a spot to run the Goroutine. It will continue
    in a loop until it does aquire also pausing if the pause flag has been
    enabled. Aquiring is throttled at the value of AquireBuffer.

func (sem *Semaphore) Release(pts int32)
    Release will release a spot for another Goroutine to take. It accepts a
    current value of remaining point balance, to which the remaining point
    balance will only be updated if the count is greater than -1. If the
    remaining points is below the set threshold, a pause will be initiated and
    a duration of this pause will be calculated based upon several factors
    surrouding the point information such as limit, threshold, and the refull
    rate.
```

## LICENSE

This project is released under the MIT [license](https://github.com/gnikyt/shopifysemaphore/blob/master/LICENSE).
