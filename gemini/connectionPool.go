package gemini

import (
	"gemini-grc/logging"
)

var IpPool IpAddressPool = IpAddressPool{IPs: make(map[string]int)}

func AddIPsToPool(IPs []string) {
	IpPool.Lock.Lock()
	for _, ip := range IPs {
		logging.LogDebug("Adding %s to pool", ip)
		IpPool.IPs[ip]++
	}
	IpPool.Lock.Unlock()
}

func RemoveIPsFromPool(IPs []string) {
	IpPool.Lock.Lock()
	for _, ip := range IPs {
		_, ok := IpPool.IPs[ip]
		if ok {
			logging.LogDebug("Removing %s from pool", ip)
			if IpPool.IPs[ip] == 1 {
				delete(IpPool.IPs, ip)
			} else {
				IpPool.IPs[ip]--
			}
		}
	}
	IpPool.Lock.Unlock()
}
