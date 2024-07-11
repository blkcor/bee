package bee

import (
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"
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
	prefix        string
	middlewares   []HandlerFunc
	parent        *RouterGroup
	engine        *Engine
	htmlTemplates *template.Template
	funcMap       template.FuncMap
}

func New() *Engine {
	engine := &Engine{
		router: newRouter(),
	}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}

func (e *Engine) SetFuncMap(funcMap template.FuncMap) {
	e.funcMap = funcMap
}

func (e *Engine) LoadHTMLGlob(pattern string) {
	e.htmlTemplates = template.Must(template.New("").Funcs(e.funcMap).ParseGlob(pattern))
}

// GET request register
func (e *Engine) GET(pattern string, handler HandlerFunc) {
	e.router.addRoute("GET", pattern, handler)
}

// POST request register
func (e *Engine) POST(pattern string, handler HandlerFunc) {
	e.router.addRoute("POST", pattern, handler)
}

// Run to start blkcor http server
func (e *Engine) Run(addr string) (err error) {
	return http.ListenAndServe(addr, e)
}

// impl the interface http.Handler
func (e *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var middlewares []HandlerFunc
	for _, group := range e.groups {
		if strings.HasPrefix(req.URL.Path, group.prefix) {
			middlewares = append(middlewares, group.middlewares...)
		}
	}
	context := newContext(w, req)
	context.handlers = middlewares
	context.engine = e
	e.router.handle(context)
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

func (rg *RouterGroup) Use(middlewares ...HandlerFunc) {
	rg.middlewares = append(rg.middlewares, middlewares...)
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

// createStaticHandler create blkcor handler to serve static files
func (rg *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := path.Join(rg.prefix, relativePath)
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))
	return func(c *Context) {
		file := c.Param("filepath")
		// Check if file exists and/or if we have permission to access it
		if _, err := fs.Open(file); err != nil {
			log.Printf("Error opening file: %s", err)
			c.Status(http.StatusNotFound)
			return
		}
		fileServer.ServeHTTP(c.Writer, c.Req)
	}
}

// Static serve static files
func (rg *RouterGroup) Static(relativePath, root string) {
	handler := rg.createStaticHandler(relativePath, http.Dir(root))
	urlPattern := path.Join(relativePath, "/*filepath")
	//register GET handler
	rg.GET(urlPattern, handler)
}
