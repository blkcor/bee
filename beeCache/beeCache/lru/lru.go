package lru

import "container/list"

type Cache struct {
	maxBytes int64
	nBytes   int64
	ll       *list.List
	cache    map[string]*list.Element
	//某条记录被移除时的回调函数
	OnEvicted func(key string, value Value)
}

type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int
}

func New(maxBytes int64, onEvicted func(key string, value Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		nBytes:    0,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Get the value of the element from the cache and update position
func (c *Cache) Get(key string) (value Value, ok bool) {
	if v, ok := c.cache[key]; ok {
		//将访问的元素移动到队尾(这里默认队首为back,队尾为front)
		c.ll.MoveToFront(v)
		kv := v.Value.(*entry)
		return kv.value, true
	}
	return
}

// RemoveOldest remove  the record from the queue and the cache
func (c *Cache) RemoveOldest() {
	if ele := c.ll.Back(); ele != nil {
		//先从队列中删除
		c.ll.Remove(ele)
		//再删除缓存中的key，并且更新缓存信息
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		c.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		//判断删除元素的回调是否注册
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Add new record to the cache
func (c *Cache) Add(key string, value Value) {
	//如果key存在在cache中，但是值有变化
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nBytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		//key不存在在cache中，直接插入
		ele := c.ll.PushFront(&entry{
			key:   key,
			value: value,
		})
		c.cache[key] = ele
		c.nBytes += int64(len(key)) + int64(value.Len())
	}
	//是否达到cache的最大值 ==> 淘汰元素
	//maxBytes为0的时候容量没有限制
	for c.maxBytes != 0 && c.nBytes > c.maxBytes {
		c.RemoveOldest()
	}
}

// Len return the length of the list
func (c *Cache) Len() int {
	return c.ll.Len()
}
