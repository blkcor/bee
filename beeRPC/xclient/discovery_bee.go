package xclient

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type BeeRegistryDiscovery struct {
	*MultiServersDiscovery
	registry   string
	timeout    time.Duration
	lastUpdate time.Time
}

const defaultUpdateTimeout = time.Second * 10

func NewBeeRegistryDiscovery(registry string, timeout time.Duration) *BeeRegistryDiscovery {
	if timeout == 0 {
		timeout = defaultUpdateTimeout
	}
	d := &BeeRegistryDiscovery{
		MultiServersDiscovery: NewMultiServerDiscovery(make([]string, 0)),
		registry:              registry,
		timeout:               timeout,
	}
	return d
}

func (brd *BeeRegistryDiscovery) Update(servers []string) error {
	brd.mu.Lock()
	defer brd.mu.Unlock()
	brd.servers = servers
	brd.lastUpdate = time.Now()
	return nil
}

func (brd *BeeRegistryDiscovery) Refresh() error {
	brd.mu.Lock()
	defer brd.mu.Unlock()
	// 如果在超时时间内，不进行更新
	if brd.lastUpdate.Add(brd.timeout).After(time.Now()) {
		return nil
	}
	log.Println("rpc registry: refresh servers from registry", brd.registry)
	resp, err := http.Get(brd.registry)
	if err != nil {
		log.Println("rpc registry refresh err:", err)
		return err
	}
	servers := strings.Split(resp.Header.Get("X-Beerpc-Servers"), ",")
	brd.servers = make([]string, 0, len(servers))
	for _, server := range servers {
		if strings.TrimSpace(server) != "" {
			brd.servers = append(brd.servers, strings.TrimSpace(server))
		}
	}
	brd.lastUpdate = time.Now()
	return nil
}

func (brd *BeeRegistryDiscovery) Get(mode SelectMode) (string, error) {
	if err := brd.Refresh(); err != nil {
		return "", err
	}
	return brd.MultiServersDiscovery.Get(mode)
}

func (brd *BeeRegistryDiscovery) GetAll() ([]string, error) {
	if err := brd.Refresh(); err != nil {
		return nil, err
	}
	return brd.MultiServersDiscovery.GetAll()
}
