package mouse_tool

import (
	"context"
	"errors"
	"fmt"
	v3 "go.etcd.io/etcd/client/v3"
	"log"
	"time"
)

const Key = "mouse:server:node:"

var (
	ErrAppendNode = errors.New("failed to register machine to etcd")
)

type EtcdClient struct {
	cli *v3.Client
	// local port
	ip   string
	port int
	quit chan struct{}
}

func NewEtcdClient(port int, endpoints []string) (*EtcdClient, error) {
	cli, err := v3.New(v3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		//log.Fatal("failed to connect etcd server, error: ", err)
		return nil, err
	}
	return &EtcdClient{cli: cli, port: port, quit: make(chan struct{})}, nil
}

func (e *EtcdClient) AppendToEtcd() error {
	// get local ip
	var err error
	if e.ip, err = GetExternalIP(); err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", e.ip, e.port)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	grant, err := e.cli.Grant(ctx, 5)
	if err != nil {
		log.Println(ErrAppendNode, err)
		return ErrAppendNode
	}
	_, err = e.cli.Put(ctx, Key+e.ip, addr, v3.WithLease(grant.ID))
	if err != nil {
		// handle error!
		log.Println(ErrAppendNode, err)
		return ErrAppendNode
	}
	go e.RentLease(grant.ID)
	return nil
}

func (e *EtcdClient) Close() {
	defer e.cli.Close()
	e.quit <- struct{}{}
}

// RentLease make a new rent for lease to avoid key expired
func (e *EtcdClient) RentLease(key v3.LeaseID) {
	// every 3 second keepalive
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-ticker.C:
			_, err := e.cli.KeepAlive(context.TODO(), key)
			if err != nil {
				log.Println("rent lease failed, please check...")
			}
		case <-e.quit:
			_, err := e.cli.Delete(context.TODO(), Key+e.ip)
			if err != nil {
				log.Println("rent lease failed, please check...")
			}
		}
	}
}
