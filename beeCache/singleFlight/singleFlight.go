package singleFlight

import "sync"

type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	// 加锁并且初始化
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}

	// 判断是否有相同的key正在被调用
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		// 存在相同的key，等待调用完成
		c.wg.Wait()
		return c.val, nil
	}

	// 不存在相同的key，新建一个call
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	//清理函数，防止内存泄漏
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
