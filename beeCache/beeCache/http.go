package beeCache

import (
	"fmt"
	"net/http"
	"strings"
)

const defaultBasePath = "/_beeCache/"

type HTTPPool struct {
	//记录http的主机 / ip + 端口
	self     string
	basePath string
}

// NewHTTPPool creates a new HTTPPool instance
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log info with server name
func (p *HTTPPool) Log(format string, v ...interface{}) {
	fmt.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	//判断前缀是否满足
	if !strings.HasPrefix(req.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + req.URL.Path)
	}
	p.Log("%s %s", req.Method, req.URL.Path)

	parts := strings.SplitN(req.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	groupName := parts[0]
	key := parts[1]
	//获取缓存组
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "No such cache group "+groupName, http.StatusNotFound)
		return
	}
	v, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	_, err = w.Write(v.ByteSlice())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
