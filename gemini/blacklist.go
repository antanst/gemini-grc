package gemini

import "gemini-grc/logging"

var Blacklist *[]string

func InBlacklist(s *Snapshot) bool {
	if Blacklist == nil {
		data := ReadLines("blacklists/domains.txt")
		Blacklist = &data
		logging.LogInfo("Loaded %d blacklisted domains", len(*Blacklist))
	}
	for _, l := range *Blacklist {
		if s.Host == l {
			return true
		}
		// if strings.HasPrefix(s.URL.String(), l) {
		// 	return true
		// }
	}
	return false
}
