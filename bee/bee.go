package bee

import (
	"net/http"
)

// HandlerFunc define the handlerFunc used by bee
type HandlerFunc func(*Context)

// Engine struct
type Engine struct {
	*RouterGroup
	router *router
	groups []*RouterGroup
}

// RouterGroup struct
type RouterGroup struct {
	prefix      string
	middlewares []HandlerFunc
	parent      *RouterGroup
	engine      *Engine
}

func New() *Engine {
	engine := &Engine{
		router: newRouter(),
	}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

func (rg *RouterGroup) Group(prefix string) *RouterGroup {
	engine := rg.engine
	newGroup := &RouterGroup{
		prefix: prefix,
		parent: rg,
		engine: engine,
	}
	//append to parent.groups
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

// addRoute add route to the RouterGroup
func (rg *RouterGroup) addRoute(method, comp string, handler HandlerFunc) {
	//v1 := route.Group("/v1"); v1.GET("/hello") => comp is /hello and the actual pattern is /v1/hello
	pattern := rg.engine.prefix + comp
	rg.engine.router.addRoute(method, pattern, handler)
}

func (rg *RouterGroup) GET(pattern string, handler HandlerFunc) {
	// Call with rg.engine
	// So we just need to add the route rule tu engine
	rg.engine.addRoute("GET", rg.prefix+pattern, handler)
}

func (rg *RouterGroup) POST(pattern string, handler HandlerFunc) {
	rg.engine.addRoute("POST", rg.prefix+pattern, handler)
}

// GET request register
func (e *Engine) GET(pattern string, handler HandlerFunc) {
	e.router.addRoute("GET", pattern, handler)
}

// POST request register
func (e *Engine) POST(pattern string, handler HandlerFunc) {
	e.router.addRoute("POST", pattern, handler)
}

// Run to start a http server
func (e *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, e)
}

// impl the interface http.Handler
func (e *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	context := newContext(w, req)
	e.router.handle(context)
}
