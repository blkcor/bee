package bee

import (
	"net/http"
	"strings"
)

// HandlerFunc define the handlerFunc used by bee
type HandlerFunc func(http.ResponseWriter, *http.Request)

// Engine struct
type Engine struct {
	router map[string]HandlerFunc
}

func New() *Engine {
	return &Engine{
		router: make(map[string]HandlerFunc),
	}
}

func (e *Engine) addRoute(method, pattern string, handler HandlerFunc) {
	key := method + "-" + pattern
	e.router[key] = handler
}

// GET request register
func (e *Engine) GET(pattern string, handler HandlerFunc) {
	e.addRoute("GET", pattern, handler)
}

// POST request register
func (e *Engine) POST(pattern string, handler HandlerFunc) {
	e.addRoute("POST", pattern, handler)
}

// Run to start a http server
func (e *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, e)
}

// impl the interface http.Handler
func (e *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	method := req.Method
	path := req.URL.Path
	//try exact match
	if handle, ok := e.router[method+"-"+path]; ok {
		handle(w, req)
		return
	}
	//try match with * sign
	for pattern, handle := range e.router {
		pathWithMethod := method + "-" + path
		if matchWildcard(pattern, pathWithMethod) {
			handle(w, req)
		}
	}
}

// 匹配通配符的辅助函数
func matchWildcard(pattern, path string) bool {
	parts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(parts) != len(pathParts) {
		return false
	}

	for i, part := range parts {
		if part == "*" {
			continue
		}
		if part != pathParts[i] {
			return false
		}
	}

	return true
}
