package bee

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type H map[string]interface{}

type Context struct {
	Req        *http.Request
	Writer     http.ResponseWriter
	Path       string
	Method     string
	Params     map[string]string
	StatusCode int
	//middleware
	handlers []HandlerFunc
	index    int
	engine   *Engine
}

func (ctx *Context) Param(key string) string {
	value, _ := ctx.Params[key]
	return value
}

func newContext(writer http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Req:    req,
		Writer: writer,
		Path:   req.URL.Path,
		Method: req.Method,
		index:  -1,
	}
}

func (ctx *Context) Next() {
	ctx.index++
	s := len(ctx.handlers)
	for ; ctx.index < s; ctx.index++ {
		ctx.handlers[ctx.index](ctx)
	}
}

// PostForm get the form value
func (ctx *Context) PostForm(key string) string {
	return ctx.Req.FormValue(key)
}

// Query get the query value
func (ctx *Context) Query(key string) string {
	return ctx.Req.URL.Query().Get(key)
}

func (ctx *Context) Fail(code int, msg string) {
	//prevent to call other handlers
	ctx.index = len(ctx.handlers)
	ctx.JSON(code, H{"message": msg})
}

// Status set the status code
func (ctx *Context) Status(code int) {
	ctx.StatusCode = code
	ctx.Writer.WriteHeader(code)
}

// SetHeader set the header
func (ctx *Context) SetHeader(key string, value string) {
	ctx.Writer.Header().Set(key, value)
}

// String set the string response
func (ctx *Context) String(code int, format string, values ...interface{}) {
	ctx.SetHeader("Content-Type", "text/plain")
	ctx.Status(code)
	ctx.Writer.Write([]byte(fmt.Sprintf(format, values...)))
}

// JSON set the json response
func (ctx *Context) JSON(code int, obj interface{}) {
	ctx.SetHeader("Content-Type", "application/json")
	ctx.Status(code)
	encoder := json.NewEncoder(ctx.Writer)
	if err := encoder.Encode(obj); err != nil {
		http.Error(ctx.Writer, err.Error(), 500)
	}
}

// Data set the data response
func (ctx *Context) Data(code int, data []byte) {
	ctx.Status(code)
	ctx.Writer.Write(data)
}

// HTML set the html response
func (ctx *Context) HTML(code int, name string, data interface{}) {
	ctx.SetHeader("Content-Type", "text/html")
	ctx.Status(code)
	if err := ctx.engine.htmlTemplates.ExecuteTemplate(ctx.Writer, name, data); err != nil {
		ctx.Fail(500, err.Error())
	}
}
