package GoRPC

import (
	"GoRPC/codec"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

type Option struct {
	CodecType   codec.CodeType
}

type Server struct{
	serviceMap sync.Map
}

type request struct {
	header      *codec.Header
	argv, replyV reflect.Value
}

type serviceMethod struct {
	svc   *service
	mType *methodType
}

var DefaultOption = &Option{
	CodecType: codec.GobCodeType,
}

func NewServer() *Server{
	return &Server{}
}

var DefaultServer = NewServer()
var invalidRequest = struct {}{}

func (server *Server)Accept(lis net.Listener) error{
	for{
		conn,err := lis.Accept()
		if err != nil{
			log.Println("rpc server: listen error")
			return err
		}
		go server.ServeConn(conn)
	}
}

func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer conn.Close()
	var opt Option

	//The Option header is default using json
	if err := json.NewDecoder(conn).Decode(&opt);err != nil {
		log.Println("rpc server: option resolve error",err)
		return
	}

	decodeF := codec.CodeType2NewCodecFuncMap[opt.CodecType]
	if decodeF == nil{
		println("rpc server : CodeType not supported",opt.CodecType)
		return
	}

	server.ServeCodec(decodeF(conn))
}

func (server *Server) ServeCodec(c codec.Codec) {
	sendLock := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	for {
		req, sm,err := server.readRequest(c)
		if err != nil {
			if req == nil {
				break
			}
			req.header.Error = err.Error()
			server.sendResponse(c, req.header, invalidRequest,sendLock)
			log.Println("server received invalid request")
			continue
		}
		wg.Add(1)
		go server.handleRequest(c,req,sm,wg,sendLock)
	}

	wg.Wait()
	err := c.Close()
	if err != nil{
		log.Println("server error:ServeCodec fail to close codec")
	}
}

func (server *Server) readRequest(c codec.Codec) (*request, *serviceMethod,error) {
	//read request header
	var h codec.Header
	var err error
	if err = c.ReadHeader(&h); err != nil{
		if err != io.EOF && err != io.ErrUnexpectedEOF{
			log.Println("rpc server: read header error",err)
		}
		return nil,nil,err
	}
	svcMethod := &serviceMethod{
		svc: nil,
		mType: nil,
	}
	req := &request{header: &h}
	//read request body
	svcMethod.svc,svcMethod.mType,err = server.findService(h.Service,h.Method)
	if err != nil {
		return req,svcMethod,err
	}


	req.argv = svcMethod.mType.newArgv()
	req.replyV = svcMethod.mType.newReplyv()

	//if argv is not a pointer, make it to pointer
	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}

	if err = c.ReadBody(argvi); err != nil{
		log.Println("rpc server: read body error")
		return req,svcMethod,err
	}
	return req,svcMethod,err
}

func (server *Server) handleRequest(c codec.Codec, req *request, sm *serviceMethod,wg *sync.WaitGroup,sendLock *sync.Mutex) {
	defer wg.Done()
	err := sm.svc.call(sm.mType,req.argv,req.replyV)
	if err != nil {
		req.header.Error = err.Error()
		server.sendResponse(c,req.header,invalidRequest,sendLock)
		return
	}
	server.sendResponse(c,req.header,req.replyV.Interface(),sendLock)
}

func (server *Server) sendResponse(c codec.Codec, header *codec.Header, body interface{},sendLock *sync.Mutex) {
	sendLock.Lock()
	defer sendLock.Unlock()
	if err := c.Write(header,body); err != nil{
		log.Println("rec server: write response error",err)
	}
}

func (server *Server) Register(rcrv interface{}) error {
	s:= newService(rcrv)
	if _,dup := server.serviceMap.LoadOrStore(s.name,s); dup {
		return errors.New("rpc: service already defined" + s.name)
	}
	return nil
}

func (server *Server) findService(serviceName, methodName string)(svc *service, mType *methodType,err error){
	svci, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc server: can't find service" + serviceName)
		return
	}
	svc = svci.(*service)
	mType = svc.method[methodName]
	if mType == nil {
		err = errors.New("rpc server: can't find method" + methodName)
	}
	return
}
