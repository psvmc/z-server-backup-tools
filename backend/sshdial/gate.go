package sshdial

import (
	"sync"
	"time"
)

const minInterval = 800 * time.Millisecond

type gate struct {
	mu       sync.Mutex
	lastDone time.Time
}

var gates sync.Map

// Acquire 串行化同一 addr 的 SSH 拨号，并在两次拨号之间保留最小间隔，
// 避免远程 sshd 因 MaxStartups / 连接风暴在握手阶段主动断开。
func Acquire(addr string) func() {
	v, _ := gates.LoadOrStore(addr, &gate{})
	g := v.(*gate)
	g.mu.Lock()
	if wait := minInterval - time.Since(g.lastDone); wait > 0 {
		time.Sleep(wait)
	}
	return func() {
		g.lastDone = time.Now()
		g.mu.Unlock()
	}
}
