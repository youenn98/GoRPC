package GoRPC

import (
	"GoRPC/codec"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
)

type Call struct {
	Seq		uint64
	Service string
	Method  string
	Args    interface{}
	Reply	interface{}
	Error   error
	Done    chan *Call		//when call is complete,
}

func (call *Call) done()  {
	call.Done <- call
}

type Client struct {
	cc     		codec.Codec
	opt    		*Option
	sendLock	sync.Mutex
	header      codec.Header
	mu    		sync.Mutex
	seq         uint64
	pending     map[uint64]*Call
	closing     bool
	shutdown    bool
}

var ErrShutdown = errors.New("connection is shut down")

func (client *Client) Close() error{
		client.mu.Lock()
		defer client.mu.Unlock()
		if client.closing{
			return ErrShutdown
		}
		client.closing = true
		return client.cc.Close()
}

func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.shutdown && !client.closing
}

func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing || client.shutdown{
		return 0,ErrShutdown
	}
	call.Seq = client.seq
	client.pending[call.Seq] = call
	client.seq++
	return call.Seq, nil
}

func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	call := client.pending[seq]
	delete(client.pending,seq)
	return call
}

func (client *Client) terminateCalls(err error){
	client.sendLock.Lock()
	defer client.sendLock.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()
	client.shutdown = true
	for _,call := range client.pending {
		call.Error = err
		call.done()
	}
}

func (client *Client) receive() {
	var err error
	for err == nil{
		var h codec.Header
		if err = client.cc.ReadHeader(&h);err != nil{
			break
		}
		call := client.removeCall(h.Seq)
		switch  {
		case call == nil:
			err = client.cc.ReadBody(nil)
		case h.Error != "":
			call.Error = fmt.Errorf(h.Error)
			err = client.cc.ReadBody(nil)
			call.done()
		default:
			err = client.cc.ReadBody(call.Reply)
			if err != nil{
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}
	client.terminateCalls(err)
}

func NewClient(conn net.Conn,opt *Option) (*Client,error){
	f := codec.CodeType2NewCodecFuncMap[opt.CodecType]
	if f == nil {
		err:= fmt.Errorf("invalid codec type %s",opt.CodecType)
		log.Println("rpc client: codec error",err)
		return nil,err
	}
	//send option to server
	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc client: options error: ",err)
		_ = conn.Close()
		return nil, err
	}
	return newClientCodec(f(conn),opt),nil
}

func newClientCodec(cc codec.Codec,opt *Option) *Client {
	client := &Client{
		seq:       1,
		cc:        cc,
		opt:       opt,
		pending:   make(map[uint64]*Call),
	}

	go client.receive()
	return client
}

func parseOptions(opts ...*Option) (*Option,error) {
	//if opts is nil or pass nil as parameter
	if len(opts) == 0 || opts[0] == nil{
		return DefaultOption, nil
	}

	if len(opts) != 1 {
		return  nil, errors.New("number of option is more than 1")
	}
	opt := opts[0]
	if opt.CodecType == ""{
		opt.CodecType = DefaultOption.CodecType
	}
	return opt,nil
}

//Dial to an RPC server
func Dial(network, address string ,opts ... *Option) (client *Client,err error){
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}
	conn, err := net.Dial(network,address)
	if err != nil{
		return nil, err
	}
	defer func() {
		if client == nil{
			_ = conn.Close()
		}
	}()
	return NewClient(conn,opt)
}

func (client *Client) send(call *Call) {
	//ensure send whole request at once, no other request interrupt
	client.sendLock.Lock()
	defer client.sendLock.Unlock()

	//register the call
	seq,err := client.registerCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}

	client.header.Service = call.Service
	client.header.Method  = call.Method
	client.header.Seq     = seq
	client.header.Error   = ""

	if err := client.cc.Write(&client.header,call.Args);err != nil {
		call := client.removeCall(seq)
		if call != nil {
			call.Error = err
			call.done()
		}
	}
}

func (client *Client) AsyCall(service string, method string, args interface{}, reply interface{}, done chan *Call) *Call{
	if done == nil{
		done = make(chan * Call,1)
	} else if cap(done) == 0{
		log.Panic("rpc client: done channel is unbuffered")
	}
	call := &Call{
		Service: service,
		Method: method,
		Args: args,
		Reply: reply,
		Done: done,
	}
	client.send(call)
	return call
}

func (client *Client)Call(service ,method string,args,reply interface{}) error {
	call := <-client.AsyCall(service,method,args,reply,make(chan *Call,1)).Done
	return call.Error
}
