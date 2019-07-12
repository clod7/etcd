package etcdService

import (
	"context"
	"go.etcd.io/etcd/clientv3"
	"log"
	"time"
)

type ServerRegister struct {
	client        *clientv3.Client
	lease         clientv3.Lease
	leaseResp     *clientv3.LeaseGrantResponse
	leaseKeepchan <-chan *clientv3.LeaseKeepAliveResponse
	cancelFunc    context.CancelFunc
	stopChan      chan struct{}
}

func NewServerRegister(addrs []string, ttl int64) *ServerRegister {
	sr := ServerRegister{}
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   addrs,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Println(err)
		return nil
	}

	sr.client = client

	sr.SetLease(ttl)
	sr.SetKeepalive()

	go sr.listenleaseKeepchan()

	return &sr
}

func (a *ServerRegister) SetLease(ttl int64) {
	lease := clientv3.NewLease(a.client)
	leaseResp, err := lease.Grant(context.TODO(), ttl)
	if err != nil {
		log.Println(err)
	}

	a.lease = lease
	a.leaseResp = leaseResp

	log.Println("start lease ", leaseResp.ID)
}

func (a *ServerRegister) SetKeepalive() {
	ctx, cancelFunc := context.WithCancel(context.Background())
	leaseKeepchan, err := a.lease.KeepAlive(ctx, a.leaseResp.ID)
	if err != nil {
		log.Println(err)
	}

	a.leaseKeepchan = leaseKeepchan
	a.cancelFunc = cancelFunc
}

func (a *ServerRegister) listenleaseKeepchan() {
	for {
		select {
		case lkchan := <-a.leaseKeepchan:
			if lkchan == nil {
				return
			}
		case <-a.stopChan:
			log.Println("stop", a.leaseResp.ID)
			a.RevokeLease()
		}
	}
}

func (a *ServerRegister) RevokeLease() {
	a.cancelFunc()

	_, err := a.client.Revoke(context.TODO(), a.leaseResp.ID)
	if err != nil {
		log.Println(err)
	}
}

func (a *ServerRegister) PutServer(key, val string) error {
	kv := clientv3.NewKV(a.client)
	_, err := kv.Put(context.TODO(), key, val, clientv3.WithLease(a.leaseResp.ID))

	return err
}

func (a *ServerRegister) Close() {
	a.stopChan <- struct{}{}
}
