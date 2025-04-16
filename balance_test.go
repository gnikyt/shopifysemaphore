package shopifysemaphore

import (
	"testing"
)

func newBalance() *Balance {
	return NewBalance(100, 1000, 100)
}

func TestRefillDuration(t *testing.T) {
	var b *Balance
	var dur string

	b = newBalance()
	dur = b.RefillDuration().String()
	if dur != "0s" {
		// Should be 0s as (1000-1000)/100 = 0.
		t.Errorf("Balance.RefillDuration() = %s; want 0s", dur)
	}

	b.Update(0)
	dur = b.RefillDuration().String()
	if dur != "10s" {
		// Should 10s as (1000-0)/100 = 10.
		t.Errorf("Balance.RefillDuration() = %s; want 0s", dur)
	}
}

func TestAtThreshold(t *testing.T) {
	var b *Balance
	var at bool

	b = newBalance()
	at = b.AtThreshold()
	if at {
		// Should not be at threshold as balance is 1000 and threshold is 100.
		t.Errorf("Balance.AtThreshold() = %v; want false", at)
	}

	b.Update(0)
	at = b.AtThreshold()
	if !at {
		// Should be at threshold as balance is 0 and threshold is 100.
		t.Errorf("Balance.AtThreshold() = %v; want true", at)
	}
}

func TestNewBalance(t *testing.T) {
	b := newBalance()
	if b.Limit != 1000 {
		t.Errorf("Balance.Limit = %d; want 1000", b.Limit)
	}
	if b.Threshold != 100 {
		t.Errorf("Balance.Threshold = %d; want 100", b.Threshold)
	}
	if b.RefillRate != 100 {
		t.Errorf("Balance.RefillRate = %d; want 100", b.RefillRate)
	}
	if b.Remaining.Load() != 1000 {
		t.Errorf("Balance.Remaining = %d; want 1000", b.Remaining.Load())
	}
}
