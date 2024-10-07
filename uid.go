package main

import (
	nanoid "github.com/jaevor/go-nanoid"
)

func UID() string {
	// Missing o,O and l
	uid, err := nanoid.CustomASCII("abcdefghijkmnpqrstuvwxyzABCDEFGHIJKLMNPQRSTUVWXYZ0123456789", 18)
	if err != nil {
		panic(err)
	}
	return uid()
}
