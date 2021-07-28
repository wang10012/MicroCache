package cache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

const defaultPrefix = "/microCache/"

type HTTPPool struct {
	selfAddress string
	prefix      string
}

func NewHTTPPool(selfAddress string) *HTTPPool {
	return &HTTPPool{
		selfAddress: selfAddress,
		prefix:      defaultPrefix,
	}
}

func (pool *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", pool.selfAddress, fmt.Sprintf(format, v...))
}

func (pool *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. 判断前缀是否是用来节点通信的前缀
	if !strings.HasPrefix(r.URL.Path, pool.prefix) {
		panic("Access Unexpected Path: " + r.URL.Path)
	}
	pool.Log("%s %s", r.Method, r.URL.Path)

	// 2. 约定访问路由格式：/<prefix>/<cacheGroupName>/<key>
	parts := strings.SplitN(r.URL.Path[len(pool.prefix):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	cacheGroupName := parts[0]
	key := parts[1]

	group := GetCacheGroup(cacheGroupName)
	if group == nil {
		http.Error(w, "No such group: "+cacheGroupName, http.StatusNotFound)
		return
	}

	onlyReadBytes, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	// 输入orb的切片，防止外部篡改
	w.Write(onlyReadBytes.ByteSlice())
}
