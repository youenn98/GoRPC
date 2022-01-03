package main

import (
	"GoRPC"
	"GoRPC/codec"
	"encoding/json"
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


	conn, _ := net.Dial("tcp",<-addr)
	defer conn.Close()

	//Send option to server
	if err := json.NewEncoder(conn).Encode(GoRPC.DefaultOption);err != nil{
		log.Fatal("Fail to send Option")
	}

	cc := codec.NewGobCodec(conn)

	wg := new(sync.WaitGroup)

	for i:=0;i < 100;i++ {
		wg.Add(1)
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

}