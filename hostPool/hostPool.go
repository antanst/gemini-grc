package hostPool

import (
	"sync"
	"time"

	"gemini-grc/logging"
)

var hostPool = HostPool{hostnames: make(map[string]struct{})} //nolint:gochecknoglobals

type HostPool struct {
	hostnames map[string]struct{}
	Lock      sync.RWMutex
}

func (p *HostPool) Add(key string) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	p.hostnames[key] = struct{}{}
}

func (p *HostPool) Get(key string) bool {
	p.Lock.RLock()
	defer p.Lock.RUnlock()
	_, ok := p.hostnames[key]
	return ok
}

func (p *HostPool) Delete(key string) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	delete(p.hostnames, key)
}

func AddHostToHostPool(key string) {
	for {
		// Sleep until the host doesn't exist in pool,
		// then add it.
		if hostPool.Get(key) {
			time.Sleep(1 * time.Second) // Avoid flood-retrying
			logging.LogInfo("Waiting to add %s to pool...", key)
		} else {
			hostPool.Add(key)
			return
		}
	}
}

func RemoveHostFromHostPool(key string) {
	if hostPool.Get(key) {
		hostPool.Delete(key)
	}
}
