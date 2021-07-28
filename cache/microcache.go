package cache

import (
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
func (group *cacheGroup) load(key string) (value OnlyReadBytes, err error) {
	return group.getFromLocal(key)
}

// 单机并发场景下，从各种数据源中加载数据到缓存
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
