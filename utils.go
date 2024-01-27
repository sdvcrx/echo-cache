package cache

import "log"

func must[T any](val T, err error) T {
	if err != nil {
		log.Fatalln(err)
	}
	return val
}

// Copy from std lib "maps"
func CopyMap[M1 ~map[K]V, M2 ~map[K]V, K comparable, V any](dst M1, src M2) {
	for k, v := range src {
		dst[k] = v
	}
}
