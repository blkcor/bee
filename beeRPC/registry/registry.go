package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// BeeRegistry is a simple register center, provide following functions.
// add a server and receive heartbeat to keep it alive.
// returns all alive servers and delete dead servers sync simultaneously.
type BeeRegistry struct {
	timeout time.Duration
	mu      sync.Mutex
	servers map[string]*ServerItem
}

type ServerItem struct {
	Addr  string
	start time.Time
}

const (
	defaultPath    = "/_beerpc_/registry"
	defaultTimeout = time.Minute * 5
)

func New(timeout time.Duration) *BeeRegistry {
	return &BeeRegistry{
		servers: make(map[string]*ServerItem),
		timeout: timeout,
	}
}

var DefaultBeeRegister = New(defaultTimeout)

// putServer adds a server to registry.
func (registry *BeeRegistry) putServer(addr string) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	s := registry.servers[addr]
	if s == nil {
		registry.servers[addr] = &ServerItem{Addr: addr, start: time.Now()}
	} else {
		s.start = time.Now() // if exists, update start time to keep alive
	}
}

// aliveServers returns a list of all alive servers
func (registry *BeeRegistry) aliveServers() []string {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	var aliveServers []string
	for addr, server := range registry.servers {
		if registry.timeout == 0 || server.start.Add(registry.timeout).After(time.Now()) {
			aliveServers = append(aliveServers, addr)
		} else {
			delete(registry.servers, addr)
		}
	}
	sort.Strings(aliveServers)
	return aliveServers
}

func (registry *BeeRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		// keep it simple, server is in req.Header
		w.Header().Set("X-Beerpc-Servers", strings.Join(registry.aliveServers(), ","))
	case "POST":
		// keep it simple, server is in req.Header
		addr := req.Header.Get("X-Beerpc-Server")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		registry.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (registry *BeeRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, registry)
	log.Println("rpc registry path:", registryPath)
}

func HandleHTTP() {
	DefaultBeeRegister.HandleHTTP(defaultPath)
}

func HeartBeats(registry, addr string, duration time.Duration) {
	if duration == 0 {
		// make sure there is enough time to send heart beat
		duration = defaultTimeout - time.Duration(1)*time.Minute
	}
	//启动一个定时器定时去发送心跳，如果发送失败，就停止发送，注册中心会自动删除这个服务
	var err error
	err = sendHeartbeat(registry, addr)
	go func() {
		t := time.NewTicker(duration)
		for err == nil {
			<-t.C
			err = sendHeartbeat(registry, addr)
		}
	}()
}

func sendHeartbeat(registry, addr string) error {
	log.Println(addr, "send heart beat to registry", registry)
	httpClient := &http.Client{}
	req, _ := http.NewRequest("POST", registry, nil)
	req.Header.Set("X-Beerpc-Server", addr)
	if _, err := httpClient.Do(req); err != nil {
		log.Println("rpc server: heart beat err:", err)
		return nil
	}
	return nil
}
