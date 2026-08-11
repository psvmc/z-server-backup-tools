package service

import (
	"fmt"
	"sync"
)

var defaultJobGate = NewJobGate()

type JobGate struct {
	mu        sync.Mutex
	multi     bool
	single    bool
	folderZip bool
}

func NewJobGate() *JobGate {
	return &JobGate{}
}

func (g *JobGate) hasDirectJob() bool {
	return g.single || g.folderZip
}

func (g *JobGate) TryAcquireMulti() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.hasDirectJob() {
		return fmt.Errorf("已有单文件或文件夹压缩备份在运行")
	}
	if g.multi {
		return fmt.Errorf("已有备份任务在运行")
	}
	g.multi = true
	return nil
}

func (g *JobGate) ReleaseMulti() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.multi = false
}

func (g *JobGate) TryAcquireSingle() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.multi {
		return fmt.Errorf("已有备份任务在运行")
	}
	if g.hasDirectJob() {
		return fmt.Errorf("已有单文件或文件夹压缩备份在运行")
	}
	g.single = true
	return nil
}

func (g *JobGate) ReleaseSingle() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.single = false
}

func (g *JobGate) TryAcquireFolderZip() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.multi {
		return fmt.Errorf("已有备份任务在运行")
	}
	if g.hasDirectJob() {
		return fmt.Errorf("已有单文件或文件夹压缩备份在运行")
	}
	g.folderZip = true
	return nil
}

func (g *JobGate) ReleaseFolderZip() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.folderZip = false
}

func (g *JobGate) MultiRunning() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.multi
}

func (g *JobGate) SingleRunning() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.single
}

func (g *JobGate) FolderZipRunning() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.folderZip
}
