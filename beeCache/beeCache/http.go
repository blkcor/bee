package beeCache

import (
	"fmt"
	"github.com/blkcor/beeCache/consistentHash"
	pb "github.com/blkcor/beeCache/proto"
	"github.com/golang/protobuf/proto"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_beeCache/"
	defaultReplicas = 50
)

type HTTPPool struct {
	//记录http的主机 / ip + 端口
	self        string
	basePath    string
	mu          sync.Mutex
	peers       *consistentHash.Map
	httpGetters map[string]*httpGetter
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
	fmt.Printf("[Server %s] %s\n", p.self, fmt.Sprintf(format, v...))
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

func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistentHash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// PickPeer picks a peer according to key
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)

type httpGetter struct {
	baseURL string
}

func (g *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	//对group和key进行编码
	u := fmt.Sprintf("%v%v/%v", g.baseURL, url.QueryEscape(in.Group), url.QueryEscape(in.Key))
	resp, err := http.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", resp.Status)
	}
	res, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}
	if err := proto.Unmarshal(res, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)

	}
	return nil
}

// 进行编译时的接口实现检查。这行代码本身不会在运行时执行任何操作，它的目的是在编译时确保 httpGetter 类型实现了 PeerGetter 接口
var _ PeerGetter = (*httpGetter)(nil)
