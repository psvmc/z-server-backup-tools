package sshdial

import (
	"sync"
	"time"
)

const (
	minInterval  = 1 * time.Second
	failCooldown = 5 * time.Second
)

type gate struct {
	mu       sync.Mutex
	lastDone time.Time
	cooldown time.Time
}

var gates sync.Map

func getGate(addr string) *gate {
	v, _ := gates.LoadOrStore(addr, &gate{})
	return v.(*gate)
}

// Acquire 串行化同一 addr 的 SSH 拨号，并在两次拨号之间保留最小间隔，
// 避免远程 sshd 因 MaxStartups / 连接风暴在握手阶段主动断开。
func Acquire(addr string) func() {
	g := getGate(addr)
	g.mu.Lock()
	now := time.Now()
	waitUntil := g.lastDone.Add(minInterval)
	if g.cooldown.After(waitUntil) {
		waitUntil = g.cooldown
	}
	if wait := waitUntil.Sub(now); wait > 0 {
		time.Sleep(wait)
	}
	return func() {
		g.lastDone = time.Now()
		g.mu.Unlock()
	}
}

// NoteFailure 在握手/拨号失败后设置冷却，给远程 sshd 恢复窗口。
func NoteFailure(addr string) {
	g := getGate(addr)
	g.mu.Lock()
	defer g.mu.Unlock()
	until := time.Now().Add(failCooldown)
	if until.After(g.cooldown) {
		g.cooldown = until
	}
}
