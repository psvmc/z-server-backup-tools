package sshdial

import (
	"sync"
	"testing"
	"time"
)

func TestAcquireSerializesAndSpacesDials(t *testing.T) {
	addr := "test.example:22"
	gates = sync.Map{}

	start := time.Now()
	release1 := Acquire(addr)
	release1()
	release2 := Acquire(addr)
	release2()
	elapsed := time.Since(start)
	if elapsed < minInterval {
		t.Fatalf("expected at least %v between dials, got %v", minInterval, elapsed)
	}
}

func TestNoteFailureAddsCooldown(t *testing.T) {
	addr := "fail.example:22"
	gates = sync.Map{}

	NoteFailure(addr)
	start := time.Now()
	release := Acquire(addr)
	release()
	elapsed := time.Since(start)
	if elapsed < failCooldown-500*time.Millisecond {
		t.Fatalf("expected cooldown near %v, got %v", failCooldown, elapsed)
	}
}
