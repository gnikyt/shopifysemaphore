package shopifysemaphore

import (
	"testing"
	"time"
)

func newBalance() *Balance {
	return NewBalance(100, 1000, 100)
}

// TestRefillDuration should ensure we calculate the correct
// duration it will take to refill the available points back to
// maximum.
func TestRefillDuration(t *testing.T) {
	b := newBalance()
	dur := b.RefillDuration()
	exdur := 0 * time.Second
	if dur != exdur {
		// Should be 0s as (1000-1000)/100 = 0.
		t.Errorf("Balance.RefillDuration() = %v; want %v", dur, exdur)
	}

	b.Update(0)
	dur = b.RefillDuration()
	exdur = 10 * time.Second
	if dur != exdur {
		// Should 10s as (1000-0)/100 = 10.
		t.Errorf("Balance.RefillDuration() = %v; want %v", dur, exdur)
	}
}

// TestAtThreshold should properly know when we are at the desired threshold.
func TestAtThreshold(t *testing.T) {
	b := newBalance()
	att := b.AtThreshold()
	if att {
		// Should not be at threshold as balance is 1000 and threshold is 100.
		t.Errorf("Balance.AtThreshold() = %v; want false", att)
	}

	b.Update(0)
	att = b.AtThreshold()
	if !att {
		// Should be at threshold as balance is 0 and threshold is 100.
		t.Errorf("Balance.AtThreshold() = %v; want true", att)
	}
}

// TestNewBalance ensures the "New" method properly accepts input
// and provides defaults.
func TestNewBalance(t *testing.T) {
	var limit int32 = 1000
	var thld int32 = 100
	var rr int32 = 100

	b := newBalance()
	if b.Limit != limit {
		t.Errorf("Balance.Limit = %d; want %d", b.Limit, limit)
	}
	if b.Threshold != thld {
		t.Errorf("Balance.Threshold = %d; want %d", b.Threshold, thld)
	}
	if b.RefillRate != rr {
		t.Errorf("Balance.RefillRate = %d; want %d", b.RefillRate, rr)
	}
	if b.Remaining.Load() != limit {
		t.Errorf("Balance.Remaining = %d; want %d", b.Remaining.Load(), limit)
	}
}

// TestUpdate should ensure we only update points above a value of ErrPts.
func TestUpdate(t *testing.T) {
	b := newBalance()
	b.Update(500)

	var expts int32 = 500
	rpts := b.Remaining.Load()
	if rpts != expts {
		t.Errorf("Balance.Remaining = %d; want %d", rpts, expts)
	}

	b.Update(ErrPts)
	rpts = b.Remaining.Load()
	if rpts != expts {
		t.Errorf("Balance.Remaining = %d; want %d", rpts, expts)
	}
}
