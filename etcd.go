package mouse_tool

import (
	"context"
	"errors"
	"fmt"
	v3 "go.etcd.io/etcd/client/v3"
	"log"
	"time"
)

// Status server status
type Status string

const (
	Ready   Status = "Ready"
	Working Status = "Working"
	Error   Status = "Error"
)

var (
	ErrRegisterToEtcd    = errors.New("failed to register machine to etcd")
	ErrUpdateServerState = errors.New("failed to update server state")
)

type MachineStatus struct {
	Addr  string `json:"addr"`
	State Status `json:"state"`
}

type EtcdClient struct {
	cli     *v3.Client
	ip      string
	leaseId v3.LeaseID
	// local port
	port int
	// key prefix
	prefix string
	quit   chan struct{}
}

func NewEtcdClient(prefix string, port int, endpoints []string) (*EtcdClient, error) {
	cli, err := v3.New(v3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		//log.Fatal("failed to connect etcd server, error: ", err)
		return nil, err
	}
	return &EtcdClient{
		cli:    cli,
		port:   port,
		quit:   make(chan struct{}),
		prefix: prefix,
	}, nil
}

func (e *EtcdClient) Register(status Status) error {
	// get local ip
	var err error
	if e.ip, err = GetExternalIP(); err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", e.ip, e.port)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	grant, err := e.cli.Grant(ctx, 8)
	if err != nil {
		log.Println(ErrRegisterToEtcd, err)
		return ErrRegisterToEtcd
	}
	e.leaseId = grant.ID
	_, err = e.cli.Put(ctx, e.prefix+":"+addr, string(status), v3.WithLease(grant.ID))
	if err != nil {
		log.Println(ErrRegisterToEtcd, err)
		return ErrRegisterToEtcd
	}
	go e.RentLease(grant.ID)
	return nil
}

// UpdateStatus update key status
func (e *EtcdClient) UpdateStatus(status Status) error {
	if e.leaseId == 0 {
		return e.Register(status)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	addr := fmt.Sprintf("%s:%d", e.ip, e.port)
	_, err := e.cli.Put(ctx, e.prefix+":"+addr, string(status), v3.WithLease(e.leaseId))
	if err != nil {
		log.Println(ErrUpdateServerState, err)
		return ErrUpdateServerState
	}
	return nil
}

func (e *EtcdClient) Close() {
	defer e.cli.Close()
	// delete key
	e.reset()
}

// reset reset node status
func (e *EtcdClient) reset() {
	e.quit <- struct{}{}
}

func (e *EtcdClient) Host() string {
	return e.ip
}

// RentLease make a new rent for lease to avoid key expired
func (e *EtcdClient) RentLease(key v3.LeaseID) {
	// every 3 second keepalive
	ticker := time.NewTicker(3 * time.Second)
	var (
		data = make(<-chan *v3.LeaseKeepAliveResponse)
		err  error
	)
	go func() {
		for {
			select {
			case <-data:
			case <-e.quit:
				return
			}
		}
	}()
	for {
		select {
		case <-ticker.C:
			// rent
			data, err = e.cli.KeepAlive(context.TODO(), key)
			if err != nil {
				log.Println("rent lease failed, please check...")
			}
		case <-e.quit:
			// remove machine
			_, err := e.cli.Delete(context.TODO(), e.prefix+":"+e.ip)
			if err != nil {
				log.Println("rent lease failed, please check...")
			}
		}
	}
}

// ListMachine machine list
func (e *EtcdClient) ListMachine() ([]*MachineStatus, error) {
	ans := make([]*MachineStatus, 0)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	keys, err := e.cli.Get(ctx, e.prefix, v3.WithPrefix())
	if err != nil {
		return ans, err
	}
	for _, kv := range keys.Kvs {
		ans = append(ans, &MachineStatus{Addr: string(kv.Key), State: Status(kv.Value)})
	}
	return ans, nil
}
