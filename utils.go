package cache

import "log"

func must[T any](val T, err error) T {
	if err != nil {
		log.Fatalln(err)
	}
	return val
}
