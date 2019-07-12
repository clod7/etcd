package etcdService

import (
	"context"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"google.golang.org/grpc/resolver"
	"log"
	"strings"
	"time"
)

const schema = "etcdService"

type ServerBuilder struct {
}

func NewServerBuilder() *ServerBuilder {
	return &ServerBuilder{}
}

// etcdService://authority1,authority2/endpoint
func (a *ServerBuilder) Build(target resolver.Target, cc resolver.ClientConn,
	opts resolver.BuildOption) (resolver.Resolver, error) {

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   strings.Split(target.Authority, ","),
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	r := &ServerResolver{
		AddrMap: make(map[string]string),
		cc:      cc,
		client:  client,
	}

	r.GetAddr(target.Endpoint)

	return r, nil
}

func (a *ServerBuilder) Scheme() string {
	return schema
}

type ServerResolver struct {
	AddrMap    map[string]string
	cc         resolver.ClientConn
	client     *clientv3.Client
	cancelFunc context.CancelFunc
}

func (a *ServerResolver) ResolveNow(resolver.ResolveNowOption) {}

func (a *ServerResolver) Close() {
	a.cancelFunc()
}

func (a *ServerResolver) GetAddr(prefix string) {
	getResp, err := a.client.Get(context.TODO(), prefix, clientv3.WithPrefix())
	if err != nil {
		log.Println(err)
	}

	for _, kv := range getResp.Kvs {
		a.AddrMap[string(kv.Key)] = string(kv.Value)
	}

	go a.watch(prefix)
	a.cc.UpdateState(resolver.State{Addresses: map2addr(a.AddrMap)})
}

func (a *ServerResolver) watch(prefix string) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	a.cancelFunc = cancelFunc
	rch := a.client.Watch(ctx, prefix, clientv3.WithPrefix())

	for {
		select {
		case n := <-rch:
			if err := n.Err(); err != nil {
				log.Println(err)
				return
			}

			for _, ev := range n.Events {
				switch ev.Type {
				case mvccpb.PUT:
					a.AddrMap[string(ev.Kv.Key)] = string(ev.Kv.Value)
				case mvccpb.DELETE:
					delete(a.AddrMap, string(ev.Kv.Key))
				}
			}

			a.cc.UpdateState(resolver.State{Addresses: map2addr(a.AddrMap)})
		}
	}
}

func map2addr(m map[string]string) []resolver.Address {
	list := make([]resolver.Address, 0)

	for _, v := range m {
		addr := resolver.Address{Addr: v}
		list = append(list, addr)
	}

	return list
}
