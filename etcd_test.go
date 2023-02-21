package mouse_tool

import (
	"log"
	"testing"
	"time"
)

func TestNewEtcdClient(t *testing.T) {
	client, err := NewEtcdClient(1000, []string{"localhost:2379"})
	if err != nil {
		log.Fatal(err)
	}
	if err = client.AppendToEtcd(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(10 * time.Second)
	client.Close()

}
