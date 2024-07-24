package codec

import "io"

type Header struct {
	ServiceMethod string //请求的服务名+方法名
	Seq           uint64 //请求序号，相当于请求的id，用来区分不同的请求
	Err           string //错误信息，客户端置为空，服务端如果发生错误，将错误信息置于此字段
}

// Codec encode and decode message
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

// NewCodecFunc is Codec Constructor, client and server could get Codec Constructor by Type
type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

// NewCodecFuncMap save all Codec Constructor
var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}
