package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/clod7/etcd/etcdService"
	msg "github.com/clod7/etcd/etcdService/example/lib"
	"google.golang.org/grpc"
)

const (
	Key     = "foo"
	MaxPort = 8089
	MinPort = 8080
)

type Hello struct{}

func (h *Hello) HelloServer(ctx context.Context, in *msg.HelloRequest) (*msg.HelloResponse, error) {
	log.Println(in)
	return &msg.HelloResponse{Ok: true}, nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	reg := etcdService.NewServerRegister([]string{"172.18.0.3:2379"}, int64(5))
	defer reg.Close()

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	randPort := r1.Intn(MaxPort-MinPort) + MinPort
	grpcAddr := fmt.Sprintf("127.0.0.1:%d", randPort)
	if err := reg.PutServer(Key, grpcAddr); err != nil {
		log.Println(err)
	}

	l, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Println(err)
	}

	grpcServer := grpc.NewServer()
	msg.RegisterHelloServiceServer(grpcServer, &Hello{})

	if err := grpcServer.Serve(l); err != nil {
		log.Println(err)
	}

}
