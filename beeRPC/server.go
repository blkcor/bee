package beeRPC

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blkcor/beeRPC/codec"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int        // MagicNumber marks this is a beeRPC request
	CodecType   codec.Type // Client may choose different Codec to encode body(CodecType)
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

// Server represents an RPC Server
type Server struct{}

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
	srv.serveCodec(f(conn))
}

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

func (srv *Server) serveCodec(cc codec.Codec) {
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
		go srv.handleRequest(cc, req, sending, wg)
	}
	//等待所有的请求处理完毕
	wg.Wait()
	_ = cc.Close()
}

// request store the request information
type request struct {
	h            *codec.Header // header of request
	argv, replyv reflect.Value // argv and replyv of request
}

// readRequest reads a single request
func (srv *Server) readRequest(cc codec.Codec) (*request, error) {
	header, err := srv.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: header}
	// TODO: now we don't know the type of request argv, just suppose it's string now
	req.argv = reflect.New(reflect.TypeOf(""))

	if err := cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err:", err)
	}
	return req, nil
}

// sendResponse sends the response for a request
func (srv *Server) sendResponse(cc codec.Codec, header *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(header, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

// handleRequest handles a single request
func (srv *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup) {
	// TODO, should call registered rpc methods to get the right replyv, just print argv and send a hello message
	defer wg.Done()
	log.Println(req.h, req.argv.Elem())
	req.replyv = reflect.ValueOf(fmt.Sprintf("beerpc resp %d", req.h.Seq))
	srv.sendResponse(cc, req.h, req.replyv.Interface(), sending)
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
