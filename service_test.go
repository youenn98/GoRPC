package GoRPC

import (
	"log"
	"reflect"
	"testing"
)

type Foo int

type Args struct {
	Num1, Num2 int
}

func (f Foo) Sum(args Args,reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func TestNewService(t *testing.T){
	var foo Foo
	s := newService(&foo)
	if len(s.method) != 1{
		log.Println("wrong service method, expect 1")
	}
	mType := s.method["Sum"]
	if mType == nil {
		log.Println("new service error")
	}
}

func TestMethodType_Call(t *testing.T){
	var foo Foo
	s := newService(&foo)
	mType := s.method["Sum"]

	argv   := mType.newArgv()
	replyv := mType.newReplyv()
	argv.Set(reflect.ValueOf(Args{Num1:1,Num2: 3}))
	_    = s.call(mType,argv,replyv)
	println( *replyv.Interface().(*int))
}