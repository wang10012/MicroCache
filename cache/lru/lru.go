package lru

import "container/list"

// Value 实现通用接口Value,可以是各种类型
type Value interface {
	// Len 实现Len()方法返回此值所占用的内存大小
	Len() int
}

// node 链表节点结构体,此结构体内与cache map中一样仍然包含key,便于操作
type node struct {
	key   string
	value Value
}

type Cache struct {
	// cache map[key](对应list节点的指针)
	cache       map[string]*list.Element
	elementList *list.List
	maxMemory   int64
	usedMemory  int64
	onRemove    func(key string, value Value)
}

// NewCache 构造函数
func NewCache(maxMemory int64, onRemove func(string, Value)) *Cache {
	return &Cache{
		maxMemory:   maxMemory,
		elementList: list.New(),
		cache:       make(map[string]*list.Element),
		onRemove:    onRemove,
	}
}

func (c *Cache) Len() int {
	return c.elementList.Len()
}

// Get 查找方法：由key查找map中的链表节点,然后将其移动到队尾
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		// TODO two line
		c.elementList.MoveToBack(ele)
		node := ele.Value.(*node)
		return node.value, true
	}
	return
}

// Eliminate 淘汰最近最少访问的节点,即队首的节点
func (c *Cache) Eliminate() {
	if ele := c.elementList.Front(); ele != nil {
		c.elementList.Remove(ele)
		node := ele.Value.(*node)
		delete(c.cache, node.key)
		c.usedMemory -= int64(len(node.key)) + int64(node.value.Len())
		if c.onRemove != nil {
			c.onRemove(node.key, node.value)
		}
	}
}

// Add 添加新节点
// 三种情况：1. 节点存在，则更新节点，移动当前节点到队尾。2. 节点不存在，则添加节点到队尾。3. 如果内存满了，需要淘汰
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.elementList.MoveToBack(ele)
		node := ele.Value.(*node)
		c.usedMemory += int64(value.Len()) - int64(node.value.Len())
		node.value = value
	} else {
		ele := c.elementList.PushBack(&node{key, value})
		c.cache[key] = ele
		c.usedMemory += int64(len(key)) + int64(value.Len())
	}
	// 一直等着，当使用内存>最大内存，则淘汰
	for c.maxMemory != 0 && c.usedMemory > c.maxMemory {
		c.Eliminate()
	}
}
