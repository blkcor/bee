package service

import (
	"go/ast"
	"log"
	"reflect"
	"sync/atomic"
)

type MethodType struct {
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
	numCalls  uint64
}

func (mt *MethodType) NumCalls() uint64 {
	return atomic.LoadUint64(&mt.numCalls)
}

func (mt *MethodType) NewArgv() reflect.Value {
	var argv reflect.Value
	// argType 可能为指针类型，也可能为值类型
	if mt.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(mt.ArgType.Elem())
	} else {
		argv = reflect.New(mt.ArgType).Elem()
	}
	return argv
}

func (mt *MethodType) NewReplyv() reflect.Value {
	// replyType 必须为指针类型
	replyv := reflect.New(mt.ReplyType.Elem())
	switch mt.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(mt.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(mt.ReplyType.Elem(), 0, 0))
	}
	return replyv
}

type Service struct {
	Name   string
	Typ    reflect.Type
	Rcvr   reflect.Value          //结构体的实例本身 即调用的第0个参数
	Method map[string]*MethodType //所有方法的集合
}

func NewService(rcvr interface{}) *Service {
	s := new(Service)
	s.Rcvr = reflect.ValueOf(rcvr)
	//reflect.Indirect 是 Go 语言反射包中的一个辅助函数，用于获取指针指向的值。
	//它可以解引用指针，类似于调用 Elem() 方法，但会自动处理指针链，直到遇到非指针类型或者空指针。
	s.Name = reflect.Indirect(reflect.ValueOf(rcvr)).Type().Name()
	s.Typ = reflect.TypeOf(rcvr)
	if !ast.IsExported(s.Name) {
		log.Fatalf("rpc server: %s is not a valid service name", s.Name)
	}
	s.registerMethods()
	return s
}

func (s *Service) registerMethods() {
	s.Method = make(map[string]*MethodType)
	for i := 0; i < s.Typ.NumMethod(); i++ {
		method := s.Typ.Method(i)
		mType := method.Type
		//入参的个数必须为3个 其中第一个参数是接收者 第二个参数是请求参数 第三个参数是返回参数
		if mType.NumIn() != 3 || mType.NumOut() != 1 {
			continue
		}
		//出参的个数必须是1个，且必须是error类型
		if mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}
		argType := mType.In(1)
		replyType := mType.In(2)
		if !isExportedOrBuiltinType(argType) || !isExportedOrBuiltinType(replyType) {
			continue
		}
		s.Method[method.Name] = &MethodType{
			method:    method,
			ArgType:   argType,
			ReplyType: replyType,
		}
		log.Printf("rpc server: register %s.%s\n", s.Name, method.Name)
	}
}

// isExportedOrBuiltinType 判断类型是否是导出的或者是Go语言内建的类型
func isExportedOrBuiltinType(t reflect.Type) bool {
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}

// Call 调用方法
func (s *Service) Call(m *MethodType, argv, replyv reflect.Value) error {
	atomic.AddUint64(&m.numCalls, 1)
	// get the method
	f := m.method.Func
	returnValues := f.Call([]reflect.Value{s.Rcvr, argv, replyv})
	if errInter := returnValues[0].Interface(); errInter != nil {
		return errInter.(error)
	}
	return nil
}
