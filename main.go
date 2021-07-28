package main

import (
	"MicroCache/cache"
	"fmt"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	cache.NewCacheGroup("scores", 2<<10, cache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[MapDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	selfAddress := "localhost:9999"
	pool := cache.NewHTTPPool(selfAddress)
	log.Println("microCache is running at", selfAddress)
	log.Fatal(http.ListenAndServe(selfAddress, pool))
}
