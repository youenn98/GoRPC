package main

import (
	"GoRPC"
	"GoRPC/codec"
	"fmt"
	"log"
	"net"
	"sync"
)

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


	//client,_ := GoRPC.Dial("tcp",<-addr)
	client,_ := GoRPC.Dial("tcp",<-addr,&GoRPC.Option{CodecType: codec.JsonCodeType})
	defer client.Close()

	//time.Sleep(time.Second)

	var wg sync.WaitGroup
	for i:=0;i < 1000;i++ {
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
	*/
}