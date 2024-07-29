package xclient

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

// SelectMode is the mode of selecting a server to send request(now only support two mode: Random and RoundRobinSelect)
type SelectMode int

const (
	RandomSelect SelectMode = iota
	RoundRobinSelect
)

type Discovery interface {
	Refresh() error                      // refresh from remote registry
	Update(servers []string) error       // update servers of remote registry manually
	Get(mode SelectMode) (string, error) // get a server according to the mode
	GetAll() ([]string, error)           // get all servers in the registry
}

// MultiServersDiscovery is a discovery for multi servers without a registry center
// user provides the server addresses explicitly instead
type MultiServersDiscovery struct {
	r       *rand.Rand   // generate random number
	mu      sync.RWMutex // protect following
	servers []string     // server addresses
	index   int          // record the selected position for robin algorithm
}

// NewMultiServerDiscovery creates a MultiServersDiscovery instance
func NewMultiServerDiscovery(servers []string) *MultiServersDiscovery {
	d := &MultiServersDiscovery{
		servers: servers,
		r:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	//robin算法每次轮询到的位置，为了避免每次从 0 开始，初始化时随机设定一个值。
	d.index = d.r.Intn(math.MaxInt32 - 1)
	return d
}

var _ Discovery = (*MultiServersDiscovery)(nil)

// Refresh doesn't make sense for MultiServersDiscovery, so ignore it
func (ms *MultiServersDiscovery) Refresh() error {
	return nil
}

// Update the servers of discovery dynamically if needed
func (ms *MultiServersDiscovery) Update(servers []string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.servers = servers
	return nil
}

// Get the server according to the mode
func (ms *MultiServersDiscovery) Get(mode SelectMode) (string, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	n := len(ms.servers)
	if n == 0 {
		return "", errors.New("rpc discovery: no available servers")
	}
	switch mode {
	case RandomSelect:
		return ms.servers[ms.r.Intn(n)], nil
	case RoundRobinSelect:
		s := ms.servers[ms.index%n]
		ms.index = (ms.index + 1) % n
		return s, nil
	default:
		return "", errors.New("rpc discovery: not supported select mode")
	}
}

func (ms *MultiServersDiscovery) GetAll() ([]string, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	servers := make([]string, len(ms.servers), len(ms.servers))
	copy(servers, ms.servers)
	return servers, nil
}
