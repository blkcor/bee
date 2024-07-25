package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blkcor/beeRPC/codec"
	"github.com/blkcor/beeRPC/server"
	"io"
	"log"
	"net"
	"sync"
)

// Call represents an active RPC.
type Call struct {
	Seq           uint64
	ServiceMethod string //format "<service>.<method>"
	Args          interface{}
	Reply         interface{}
	Error         error
	Done          chan *Call // for rpc handler to notify the client
}

// done call the done method to notify the client
func (call *Call) done() {
	call.Done <- call
}

// Client represents an RPC Client.
// There may be multiple outstanding Calls associated
// with a single Client, and a Client may be used by
// multiple goroutines simultaneously.
type Client struct {
	cc       codec.Codec
	opt      *server.Option
	sending  sync.Mutex //protect following
	header   codec.Header
	mu       sync.Mutex //protect following
	seq      uint64
	pending  map[uint64]*Call
	closing  bool // user has called Close
	shutdown bool // server has told us to stop
}

var _ io.Closer = (*Client)(nil)

var ErrShutdown = errors.New("connection is shut down")

// Close the connection
func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing {
		return ErrShutdown
	}
	client.closing = true
	return client.cc.Close()
}

// IsAvailable check if the connection is available
func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.shutdown && !client.closing
}

// registerCall register the call
func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing || client.shutdown {
		return 0, ErrShutdown
	}
	call.Seq = client.seq
	client.pending[call.Seq] = call
	client.seq++
	return call.Seq, nil
}

// removeCall remove the call from pending
func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing || client.shutdown {
		return nil
	}
	call := client.pending[seq]
	delete(client.pending, seq)
	return call
}

// terminateCalls terminate all calls on the client
func (client *Client) terminateCalls(err error) {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()
	client.shutdown = true
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}

// client 最重要的两个功能是发送请求和接收响应。先看接受响应部分。
// 1、call不存在，可能是请求没有发送完整，或者因为其他原因被取消，但是服务端仍旧处理了。
// 2、call存在但是err不为空，说明服务端处理请求出错。
// 3、call存在并且err为空，说明服务端处理请求成功，将响应写入call.Reply。
func (client *Client) receive() {
	var err error
	for err == nil {
		var header codec.Header
		if err = client.cc.ReadHeader(&header); err != nil {
			break
		}
		call := client.removeCall(header.Seq)
		switch {
		case call == nil:
			// it usually means that Write partially failed
			// and call was already removed.
			err = client.cc.ReadBody(nil)
		case header.Err != "":
			call.Error = errors.New(header.Err)
			err = client.cc.ReadBody(nil)
			call.done()
		default:
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}
	client.terminateCalls(err)
}

func NewClient(conn net.Conn, opt *server.Option) (*Client, error) {
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		err := fmt.Errorf("invalid codec type %s", opt.CodecType)
		log.Println("rpc client: codec error:", err)
		return nil, err
	}
	//send option with server
	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc client: options error: ", err)
		_ = conn.Close()
		return nil, err
	}
	return newClientCodec(f(conn), opt), nil
}

func newClientCodec(cc codec.Codec, opt *server.Option) *Client {
	client := &Client{
		seq:     1, // seq starts with 1, 0 means invalid call
		cc:      cc,
		opt:     opt,
		pending: make(map[uint64]*Call),
	}
	go client.receive()
	return client
}

func parseOptions(opts ...*server.Option) (*server.Option, error) {
	// if opts is nil or pass a nil parameter
	if len(opts) == 0 || opts[0] == nil {
		return server.DefaultOption, nil
	}
	if len(opts) != 1 {
		return nil, errors.New("number of options is more than 1")
	}
	opt := opts[0]
	opt.MagicNumber = server.DefaultOption.MagicNumber
	if opt.CodecType == "" {
		opt.CodecType = server.DefaultOption.CodecType
	}
	return opt, nil
}

// Dial connects to an RPC server at the specified network address.
func Dial(network, address string, opts ...*server.Option) (client *Client, err error) {
	option, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	defer func() {
		if client == nil {
			_ = conn.Close()
		}
	}()
	return NewClient(conn, option)
}

func (client *Client) send(call *Call) {
	client.sending.Lock()
	defer client.sending.Unlock()

	//register the call
	seq, err := client.registerCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}
	//prepare request header
	client.header.Seq = seq
	client.header.ServiceMethod = call.ServiceMethod
	client.header.Err = ""

	//encode and send the request
	if err := client.cc.Write(&client.header, call.Args); err != nil {
		call = client.removeCall(seq)
		// client has received the response and handled
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

// Go invokes the function asynchronously.
// It returns the Call structure representing the invocation.
func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panic("rpc client: done channel is unbuffered")
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}
	client.send(call)
	return call
}

// Call invokes the named function, waits for it to complete,
// and returns its error status.
func (client *Client) Call(serviceMethod string, args, reply interface{}) error {
	call := <-client.Go(serviceMethod, args, reply, make(chan *Call, 1)).Done
	return call.Error
}
