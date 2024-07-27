package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blkcor/beeRPC/codec"
	"github.com/blkcor/beeRPC/service"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
)

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int        // MagicNumber marks this is a beeRPC request
	CodecType   codec.Type // Client may choose different Codec to encode body(CodecType)
	//超时处理
	//1、客户端建立连接时 2、客户端 Client.Call() 整个过程导致的超时（包含发送报文，等待处理，接收报文所有阶段）
	//3、服务端处理报文，即 Server.handleRequest 超时。
	ConnectTimeout time.Duration
	HandleTimeout  time.Duration
}

var DefaultOption = &Option{
	MagicNumber:    MagicNumber,
	CodecType:      codec.GobType,
	ConnectTimeout: time.Second * 10,
}

// Server represents an RPC Server
type Server struct {
	serviceMap sync.Map
}

// NewServer returns a new Server
func NewServer() *Server {
	return &Server{}
}

// DefaultServer is the default instance of *Server
var DefaultServer = NewServer()

// Accept accepts connections on the listener and serves requests for each incoming connection
func (srv *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		//	initiate a goroutine to server connection
		go srv.ServerConn(conn)
	}
}

// ServerConn runs the server on a single connection,
// ServeConn blocks, serving the connection until the client hangs up.
func (srv *Server) ServerConn(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	// Option使用json进行编码和解码
	var option Option
	// 阻塞等待客户端发送Option
	if err := json.NewDecoder(conn).Decode(&option); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	if option.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number %x", option.MagicNumber)
		return
	}

	f := codec.NewCodecFuncMap[option.CodecType]
	if f == nil {
		log.Printf("rpc server: invalid codec type %s", option.CodecType)
		return
	}
	srv.serveCodec(f(conn), option.HandleTimeout)
}

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

func (srv *Server) serveCodec(cc codec.Codec, handleTimeout time.Duration) {
	sending := new(sync.Mutex) //make sure to send a complete response
	wg := new(sync.WaitGroup)  //wait until all request are handled
	for {
		req, err := srv.readRequest(cc)
		if err != nil {
			// server处理完header，没有新的header，服务器关闭连接
			if req == nil {
				break
			}
			req.h.Err = err.Error()
			srv.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		go srv.handleRequest(cc, req, sending, wg, handleTimeout)
	}
	//等待所有的请求处理完毕
	wg.Wait()
	_ = cc.Close()
}

// Register publishes in the server the set of methods of the
func (srv *Server) Register(rcvr interface{}) error {
	s := service.NewService(rcvr)
	if _, dup := srv.serviceMap.LoadOrStore(s.Name, s); dup {
		return errors.New("rpc: service already defined: " + s.Name)
	}
	return nil
}

func (srv *Server) findService(serviceMethod string) (svc *service.Service, mtype *service.MethodType, err error) {
	dot := strings.LastIndex(serviceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc server: service/method request ill-formed: " + serviceMethod)
		return
	}
	serviceName, methodName := serviceMethod[:dot], serviceMethod[dot+1:]
	svci, ok := srv.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc server: can't find service " + serviceName)
		return
	}
	svc = svci.(*service.Service)
	mtype = svc.Method[methodName]
	if mtype == nil {
		err = errors.New("rpc server: can't find method " + methodName)
	}
	return
}

// Register publishes the receiver's methods in the DefaultServer.
func Register(rcvr interface{}) error {
	return DefaultServer.Register(rcvr)
}

// request store the request information
type request struct {
	h            *codec.Header // header of request
	argv, replyv reflect.Value // argv and replyv of request
	mtype        *service.MethodType
	svc          *service.Service
}

// readRequest reads a single request
func (srv *Server) readRequest(cc codec.Codec) (*request, error) {
	header, err := srv.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: header}
	req.svc, req.mtype, err = srv.findService(header.ServiceMethod)
	if err != nil {
		return req, err
	}
	req.argv = req.mtype.NewArgv()
	req.replyv = req.mtype.NewReplyv()
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}
	if err := cc.ReadBody(argvi); err != nil {
		log.Println("rpc server: read argv err:", err)
		return req, err
	}
	return req, nil
}

// sendResponse sends the response for a request
func (srv *Server) sendResponse(cc codec.Codec, header *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	// 请求和响应共用一个header
	if err := cc.Write(header, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

// handleRequest handles a single request
func (srv *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup, timeout time.Duration) {
	defer wg.Done()
	called := make(chan struct{})
	sent := make(chan struct{})
	go func() {
		err := req.svc.Call(req.mtype, req.argv, req.replyv)
		called <- struct{}{}
		if err != nil {
			req.h.Err = err.Error()
			srv.sendResponse(cc, req.h, invalidRequest, sending)
			sent <- struct{}{}
			return
		}
		srv.sendResponse(cc, req.h, req.replyv.Interface(), sending)
		sent <- struct{}{}
	}()
	//没有限制超时时间
	if timeout == 0 {
		<-called
		<-sent
		return
	}

	select {
	case <-time.After(timeout):
		req.h.Err = fmt.Sprintf("rpc server: request handle timeout: expect within %s", timeout)
		srv.sendResponse(cc, req.h, invalidRequest, sending)
	case <-called:
		<-sent
	}
}

// readRequestHeader reads a request header
func (srv *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && !errors.Is(err, io.ErrUnexpectedEOF) {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}
