package uid

import (
	nanoid "github.com/jaevor/go-nanoid"
)

func UID() string {
	// Missing o,O and l
	uid, err := nanoid.CustomASCII("abcdefghijkmnpqrstuvwxyzABCDEFGHIJKLMNPQRSTUVWXYZ0123456789", 20)
	if err != nil {
		panic(err)
	}
	return uid()
}
