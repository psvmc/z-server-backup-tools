package service

import "testing"

func TestJobGateMutex(t *testing.T) {
	g := NewJobGate()
	if err := g.TryAcquireMulti(); err != nil {
		t.Fatal(err)
	}
	if err := g.TryAcquireSingle(); err == nil {
		t.Fatal("expected conflict with multi")
	}
	if err := g.TryAcquireFolderZip(); err == nil {
		t.Fatal("expected conflict with multi")
	}
	g.ReleaseMulti()
	if err := g.TryAcquireSingle(); err != nil {
		t.Fatal(err)
	}
	if err := g.TryAcquireFolderZip(); err == nil {
		t.Fatal("expected conflict with single")
	}
}
