package cache

import (
	singlefilght "MicroCache/cache/singleflight"
	"fmt"
	"log"
	"sync"
)

// Getter 接口：根据key加载数据到内存
type Getter interface {
	// Get 在这里，返回[]byte类型,而不是抽象封装过的onlyReadBytes,是为了方便用户去实现每个数据源的get函数
	Get(key string) ([]byte, error)
}

// GetterFunc 函数类型实现 Getter 接口
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

type cacheGroup struct {
	name     string
	getter   Getter
	conCache concurrentCache
	peers    PeerPicker
	// singleflight:Prevents cache breakdown
	caller *singlefilght.Group
}

var (
	rwMutex     sync.RWMutex
	cacheGroups = make(map[string]*cacheGroup)
)

// NewCacheGroup 创建cacheGroup
func NewCacheGroup(name string, cacheBytes int64, getter Getter) *cacheGroup {
	if getter == nil {
		panic("nil Getter")
	}
	rwMutex.Lock()
	defer rwMutex.Unlock()
	group := &cacheGroup{
		name:     name,
		getter:   getter,
		conCache: concurrentCache{cacheBytes: cacheBytes},
		caller:   &singlefilght.Group{},
	}
	cacheGroups[name] = group
	return group
}

// GetCacheGroup 返回相应名字的cacheGroup
// 无写操作，故只使用 只读锁
func GetCacheGroup(name string) *cacheGroup {
	rwMutex.RLock()
	group := cacheGroups[name]
	rwMutex.RUnlock()
	return group
}

// Get 暂时实现以下功能：
// 1. key在缓存中的时候返回缓存值
// 2. 不在缓存中，调用加载函数
func (group *cacheGroup) Get(key string) (OnlyReadBytes, error) {
	if key == "" {
		return OnlyReadBytes{}, fmt.Errorf("需要填写key")
	}

	if value, ok := group.conCache.get(key); ok {
		log.Println("命中缓存！")
		return value, nil
	}

	// 加载函数
	return group.load(key)
}

// 1. 单机并发的场景下，调用getFromLocal
// 2. 现在添加分布式节点场景下的逻辑
// 调用 PickPeer() 方法选择节点，若非本机节点，则调用 getFromPeer() 从远程获取。若是本机节点或失败，则回退到 调用 getFromLocal()。
func (group *cacheGroup) load(key string) (value OnlyReadBytes, err error) {
	val, err := group.caller.Do(key, func() (interface{}, error) {
		if group.peers != nil {
			if peer, ok := group.peers.PickPeer(key); ok {
				if value, err = group.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[Cache] Failed to get from peer", err)
			}
		}

		return group.getFromLocal(key)
	})
	if err == nil {
		return val.(OnlyReadBytes), err
	}
	return
}

// 从各种数据源中加载数据到缓存
func (group *cacheGroup) getFromLocal(key string) (OnlyReadBytes, error) {
	// 对于不同的数据源，get函数不同
	bytes, err := group.getter.Get(key)
	if err != nil {
		return OnlyReadBytes{}, err
	}
	value := OnlyReadBytes{b: cloneBytes(bytes)}
	group.loadIntoCache(key, value)
	return value, nil
}

// 加载到缓存
func (group *cacheGroup) loadIntoCache(key string, value OnlyReadBytes) {
	group.conCache.add(key, value)
}

// RegisterPeers 注册一个 PeerPicker 来挑选远程节点
func (group *cacheGroup) RegisterPeers(peers PeerPicker) {
	if group.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	group.peers = peers
}

// getFromPeer
func (group *cacheGroup) getFromPeer(peer PeerGetter, key string) (OnlyReadBytes, error) {
	bytes, err := peer.Get(group.name, key)
	if err != nil {
		return OnlyReadBytes{}, err
	}
	return OnlyReadBytes{b: bytes}, nil
}
