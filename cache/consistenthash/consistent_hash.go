package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash 允许替换为自定义
type Hash func(data []byte) uint32

type ConsistentHash struct {
	hash     Hash
	hashRing []int
	// key:虚拟节点哈希值 value:真实节点名称
	hashMap         map[int]string
	numVirtualNodes int
}

func NewConsistentHash(fn Hash, numVirtualNodes int) *ConsistentHash {
	obj := &ConsistentHash{
		hash:            fn,
		hashMap:         make(map[int]string),
		numVirtualNodes: numVirtualNodes,
	}
	if obj.hash == nil {
		obj.hash = crc32.ChecksumIEEE
	}
	return obj
}

// Add 添加节点
func (ch *ConsistentHash) Add(keys ...string) {
	// 对于每个真实节点，创建numVirtualNodes个虚拟节点
	for _, key := range keys {
		for i := 0; i < ch.numVirtualNodes; i++ {
			hash := int(ch.hash([]byte(strconv.Itoa(i) + key)))
			ch.hashRing = append(ch.hashRing, hash)
			ch.hashMap[hash] = key
		}
	}
	// 环上的节点排序
	sort.Ints(ch.hashRing)
}

// Get 访问环上最近的点
func (ch *ConsistentHash) Get(key string) string {
	if len(ch.hashRing) == 0 {
		return ""
	}

	hash := int(ch.hash([]byte(key)))
	// Search returns the first true index. If there is no such index, Search returns n.
	idx := sort.Search(len(ch.hashRing), func(i int) bool {
		return ch.hashRing[i] >= hash
	})

	return ch.hashMap[ch.hashRing[idx%len(ch.hashRing)]]
}
