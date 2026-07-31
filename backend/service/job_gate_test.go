package service

import "testing"

func TestJobGateMutex(t *testing.T) {
	g := NewJobGate()
	if err := g.TryAcquireMulti(); err != nil {
		t.Fatal(err)
	}
	if err := g.TryAcquireSingle(); err == nil {
		t.Fatal("expected conflict")
	}
	g.ReleaseMulti()
	if err := g.TryAcquireSingle(); err != nil {
		t.Fatal(err)
	}
}
