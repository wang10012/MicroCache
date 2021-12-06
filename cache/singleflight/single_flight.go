package singlefilght

import "sync"

type Group struct {
	mu sync.Mutex       // protect m
	m  map[string]*Call // key: call
}

// Call calling or called
type Call struct {
	wg  sync.WaitGroup
	val interface{} // res
	err error       // error
}

func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {

	g.mu.Lock()
	// Lazy load
	if g.m == nil {
		g.m = make(map[string]*Call)
	}

	// Wait if the func is calling
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	// Init a call
	c := new(Call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	// Exec th func,get res and err
	c.val, c.err = fn()
	c.wg.Done()

	// Once the func is done, delete key in map
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err

}
