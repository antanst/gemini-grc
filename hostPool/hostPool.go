package hostPool

import (
	"sync"
	"time"

	"gemini-grc/logging"
)

var hostPool = HostPool{hostnames: make(map[string]struct{})} //nolint:gochecknoglobals

type HostPool struct {
	hostnames map[string]struct{}
	lock      sync.RWMutex
}

//func (p *HostPool) add(key string) {
//	p.lock.Lock()
//	defer p.lock.Unlock()
//	p.hostnames[key] = struct{}{}
//}
//
//func (p *HostPool) has(key string) bool {
//	p.lock.RLock()
//	defer p.lock.RUnlock()
//	_, ok := p.hostnames[key]
//	return ok
//}

func RemoveHostFromPool(key string) {
	hostPool.lock.Lock()
	defer hostPool.lock.Unlock()
	delete(hostPool.hostnames, key)
}

func AddHostToHostPool(key string) {
	for {
		hostPool.lock.Lock()
		_, exists := hostPool.hostnames[key]
		if !exists {
			hostPool.hostnames[key] = struct{}{}
			hostPool.lock.Unlock()
			return
		}
		hostPool.lock.Unlock()
		time.Sleep(1 * time.Second)
		logging.LogInfo("Waiting to add %s to pool...", key)
	}
}
