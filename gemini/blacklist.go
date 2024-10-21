package gemini

import "strings"

var Blacklist *[]string

func InBlacklist(s *Snapshot) bool {
	if Blacklist == nil {
		data := ReadLines("blacklist.txt")
		Blacklist = &data
	}
	for _, l := range *Blacklist {
		if strings.HasPrefix(s.URL.String(), l) {
			return true
		}
	}
	return false
}
