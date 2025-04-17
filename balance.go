package shopifysemaphore

import (
	"sync/atomic"
	"time"
)

// ErrPts is the points value to pass in if a network or other error happens.
// Essentially to be used for situations where no response containing point
// information was returned. This is used to know if the Update method should
// actually update the remaining point balance or not.
var ErrPts int32 = -1

// Balance represents the information of point values and keeps track of
// items such as the remaining points, threshold, limit, and refill rate.
type Balance struct {
	Remaining  atomic.Int32 // Point balance remaining.
	Threshold  int32        // Minimum point balance where we would consider handling with a "pause".
	Limit      int32        // Maximum points available.
	RefillRate int32        // Number of points refilled per second.
}

// NewBalance accepts a threshold (thld) point balance, a maximum (max) point
// balance, and the refill rate (rr). It will return a pointer to Balance.
func NewBalance(thld int32, max int32, rr int32) *Balance {
	b := &Balance{
		Threshold:  thld,
		Limit:      max,
		RefillRate: rr,
	}
	b.Update(max)
	return b
}

// Update accepts a new value of remaining points to store.
func (b *Balance) Update(points int32) {
	if points > ErrPts {
		b.Remaining.Store(points)
	}
}

// RefillDuration accounts for the remaining points, the limit, and the refill rate to
// determine how many seconds it would take to refill to remaining points back to full.
// It will return a duration which can be used to "pause" operations.
func (b *Balance) RefillDuration() time.Duration {
	return time.Duration((b.Limit-b.Remaining.Load())/b.RefillRate) * time.Second
}

// AtThreshold will return a boolean if we have reached or surpassed the set
// threshold of remaining points or not.
func (b *Balance) AtThreshold() bool {
	return b.Remaining.Load() <= b.Threshold
}
