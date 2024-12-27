package gemini

import (
	"gemini-grc/logging"
)

var IPPool = IpAddressPool{IPs: make(map[string]int)}

func AddIPsToPool(ips []string) {
	IPPool.Lock.Lock()
	for _, ip := range ips {
		logging.LogDebug("Adding %s to pool", ip)
		IPPool.IPs[ip] = 1
	}
	IPPool.Lock.Unlock()
}

func RemoveIPsFromPool(IPs []string) {
	IPPool.Lock.Lock()
	for _, ip := range IPs {
		_, ok := IPPool.IPs[ip]
		if ok {
			logging.LogDebug("Removing %s from pool", ip)
			delete(IPPool.IPs, ip)
		}
	}
	IPPool.Lock.Unlock()
}
