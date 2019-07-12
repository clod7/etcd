package main

import (
	"context"
	"google.golang.org/grpc/resolver"
	"log"
	"time"

	"github.com/clod7/etcd/etcdService"
	msg "github.com/clod7/etcd/etcdService/example/lib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	b := etcdService.NewServerBuilder()
	resolver.Register(b)

	endPoint := "etcdService://172.18.0.3:2379/foo"

	conn, err := grpc.Dial(endPoint, grpc.WithInsecure(), grpc.WithBalancerName(roundrobin.Name))
	if err != nil {
		log.Println(err)
	}
	defer conn.Close()

	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	for {
		client := msg.NewHelloServiceClient(conn)

		resp, err := client.HelloServer(ctx, &msg.HelloRequest{
			Text: "hello",
		})
		if err != nil {
			log.Println(err)
		}

		log.Println(resp)

		time.Sleep(3 * time.Second)
	}

}
