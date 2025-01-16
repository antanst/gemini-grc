package gemini

import (
	"sync"
)

var ipPool = IpAddressPool{IPs: make(map[string]int)}         //nolint:gochecknoglobals
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
