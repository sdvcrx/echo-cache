package cache

import "log"

func Must[T any](val T, err error) T {
	if err != nil {
		log.Fatalln(err)
	}
	return val
}
