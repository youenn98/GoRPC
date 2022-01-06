package main

import (
	"GoRPC"
	"log"
	"net"
	"sync"
	"time"
)

/*
func tickServer(addr chan string){
	lis, err := net.Listen("tcp",":0")
	if err != nil{
		log.Fatal("tickServer error",err)
	}
	log.Println("start rpc server on",lis.Addr())
	addr <- lis.Addr().String()
	GoRPC.DefaultServer.Accept(lis)
}

func main(){
	addr := make(chan string)
	go tickServer(addr)

	netAddr := <- addr

	//client,_ := GoRPC.Dial("tcp",<-addr)
	client,_ := GoRPC.Dial("tcp",netAddr,&GoRPC.Option{CodecType: codec.JsonCodeType})
	client2,_ := GoRPC.Dial("tcp",netAddr,&GoRPC.Option{CodecType: codec.JsonCodeType})

	//defer client.Close()
	//defer client2.Close()
	//time.Sleep(time.Second)

	var wg sync.WaitGroup
	defer client.Close()
	for i:=0;i < 10;i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := fmt.Sprintf("GoRPC req %d",i)
			var reply string
			if err := client.Call("Foo","Sum",args,&reply);err != nil{
				log.Fatal("call foo.Sum error:",err)
			}
			log.Println("reply:",reply)
		}(i)
	}

	for i:=0;i < 10;i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := fmt.Sprintf("GoRPC req %d",i)
			var reply string
			if err := client2.Call("Foo","Minus",args,&reply);err != nil{
				log.Fatal("call foo.Sum error:",err)
			}
			log.Println("reply:",reply)
		}(i)
	}

	wg.Wait()



	/*
	//Send option to server
	if err := json.NewEncoder(conn).Encode(GoRPC.DefaultOption);err != nil{
		log.Fatal("Fail to send Option")
	}

	cc := codec.NewGobCodec(conn)
	for i:=0;i < 100;i++ {
		h := &codec.Header{
			Service : "Localhost",
			Method  : "echo",
			Seq     : uint64(i),
		}

		_ = cc.Write(h,fmt.Sprintf("GoRPC req %d",h.Seq))
		_ = cc.ReadHeader(h)
		var reply string
		_ = cc.ReadBody(&reply)
		log.Println("reply:",reply)
	}

}*/

type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func tickServer(addr chan string) {
	var foo Foo
	if err := GoRPC.DefaultServer.Register(&foo); err != nil {
		log.Fatal("register error:", err)
	}
	// pick a free port
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	GoRPC.DefaultServer.Accept(l)
}

func main() {
	log.SetFlags(0)
	addr := make(chan string)
	go tickServer(addr)
	client, _ := GoRPC.Dial("tcp", <-addr)
	defer func() { _ = client.Close() }()

	time.Sleep(time.Second)
	// send request & receive response
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := &Args{Num1: i, Num2: i * i}
			var reply int
			if err := client.Call("Foo","Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Printf("%d + %d = %d", args.Num1, args.Num2, reply)
		}(i)
	}
	wg.Wait()
}