package GoRPC

import (
	"GoRPC/codec"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

type Option struct {
	CodecType   codec.CodeType
}

type Server struct{}

type request struct {
	header      *codec.Header
	argv, replyV reflect.Value
}

var DefaultOption = &Option{
	CodecType: codec.GobCodeType,
}

func NewServer() *Server{
	return &Server{}
}

var DefaultServer = NewServer()
var invalidRequest = struct {

}{}

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

	decodeF := codec.CodeType2FuncMap[opt.CodecType]
	if decodeF == nil{
		println("rpc server : CodeType not supported",opt.CodecType)
		return
	}

	server.ServeCodec(decodeF(conn))
}

func (server *Server) ServeCodec(c codec.Codec) {
	wg := new(sync.WaitGroup)
	for {
		req, err := server.readRequest(c)
		if err != nil {
			if req == nil {
				break
			}
			req.header.Error = err.Error()
			server.sendResponse(c, req.header, invalidRequest)
			log.Println("server received invalid request")
			continue
		}
		wg.Add(1)
		go server.handleRequest(c,req,wg)
	}

	wg.Wait()
	err := c.Close()
	if err != nil{
		log.Println("server error:ServeCodec fail to close codec")
	}
}

func (server *Server) readRequest(c codec.Codec) (*request, error) {
	//read request header
	var h codec.Header
	if err := c.ReadHeader(&h); err != nil{
		if err != io.EOF && err != io.ErrUnexpectedEOF{
			log.Println("rpc server: read header error",err)
		}
		return nil, err
	}

	req := &request{header: &h}
	//read request body
	//Suppose to string, req.argv is a pointer to string
	req.argv = reflect.New(reflect.TypeOf(""))
	if err := c.ReadBody(req.argv.Interface()); err != nil{
		log.Println("rpc server: read body error")
		return nil,err
	}

	return req,nil

}

func (server *Server) handleRequest(c codec.Codec, req *request, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Println(req.header,req.argv.Elem())
	req.replyV = reflect.ValueOf(fmt.Sprintf("GoRPC resp: %d",req.header.Seq))
	server.sendResponse(c,req.header,req.replyV.Interface())
}

func (server *Server) sendResponse(c codec.Codec, header *codec.Header, body interface{}) {
	if err := c.Write(header,body); err != nil{
		log.Println("rec server: write response error",err)
	}
}