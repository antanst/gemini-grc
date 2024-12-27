package gemini

import "sync"

// Used to limit requests per
// IP address. Maps IP address
// to number of active connections.
type IpAddressPool struct {
	IPs  map[string]int
	Lock sync.RWMutex
}

func (p *IpAddressPool) Set(key string, value int) {
	p.Lock.Lock()         // Lock for writing
	defer p.Lock.Unlock() // Ensure mutex is unlocked after the write
	p.IPs[key] = value
}

func (p *IpAddressPool) Get(key string) int {
	p.Lock.RLock()         // Lock for reading
	defer p.Lock.RUnlock() // Ensure mutex is unlocked after reading
	if value, ok := p.IPs[key]; !ok {
		return 0
	} else {
		return value
	}
}

func (p *IpAddressPool) Delete(key string) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	delete(p.IPs, key)
}

func (p *IpAddressPool) Incr(key string) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	if _, ok := p.IPs[key]; !ok {
		p.IPs[key] = 1
	} else {
		p.IPs[key] = p.IPs[key] + 1
	}
}

func (p *IpAddressPool) Decr(key string) {
	p.Lock.Lock()
	defer p.Lock.Unlock()
	if val, ok := p.IPs[key]; ok {
		p.IPs[key] = val - 1
		if p.IPs[key] == 0 {
			delete(p.IPs, key)
		}
	}
}
