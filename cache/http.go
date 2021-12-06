package cache

import (
	"MicroCache/cache/consistenthash"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultPrefix          = "/microCache/"
	defaultNumVirtualNodes = 50
)

// HTTPPool 实现 PeerPicker 接口
type HTTPPool struct {
	selfAddress string
	prefix      string
	// add
	mu sync.Mutex
	// 用于选择节点
	peers *consistenthash.ConsistentHash
	// 每一个节点对应一个httpClient key为selfAddress:http://10.0.0.2:8234类似形式
	httpClients map[string]*httpClient
}

type httpClient struct {
	accessUrl string
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

func (h *httpClient) Get(cacheGroup string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.accessUrl,
		url.QueryEscape(cacheGroup),
		url.QueryEscape(key),
	)
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server return err: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body err: %v", err)
	}

	return bytes, nil
}

// 实现 PeerGetter 接口 断言
var _ PeerGetter = (*httpClient)(nil)

// SetNewConsistentHash 传入peers
func (p *HTTPPool) SetNewConsistentHash(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.NewConsistentHash(nil, defaultNumVirtualNodes)
	p.peers.Add(peers...)
	p.httpClients = make(map[string]*httpClient, len(peers))
	for _, peer := range peers {
		p.httpClients[peer] = &httpClient{accessUrl: peer + p.prefix}
	}
}

// PickPeer HTTPPool 实现PickPeer方法以实现PeerPicker接口
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.selfAddress {
		p.Log("Pick peer %s", peer)
		return p.httpClients[peer], true
	}
	return nil, false
}

// 实现 PeerPicker 接口 断言判定
var _ PeerPicker = (*HTTPPool)(nil)
