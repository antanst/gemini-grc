package uid

import (
	nanoid "github.com/matoous/go-nanoid/v2"
)

func UID() string {
	// No 'o','O' and 'l'
	id, err := nanoid.Generate("abcdefghijkmnpqrstuvwxyzABCDEFGHIJKLMNPQRSTUVWXYZ0123456789", 20)
	if err != nil {
		panic(err)
	}
	return id
}
