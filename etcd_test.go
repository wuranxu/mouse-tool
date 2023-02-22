package mouse_tool

import (
	"log"
	"testing"
	"time"
)

func TestNewEtcdClient(t *testing.T) {
	client, err := NewEtcdClient("mouse:server", 1000, []string{"localhost:2379"})
	if err != nil {
		log.Fatal(err)
	}
	if err = client.Register(Ready); err != nil {
		log.Fatal(err)
	}
	machine, err := client.ListMachine()
	if err != nil {
		t.Error("query failed: ", err)
	}
	t.Log(machine)
	time.Sleep(10 * time.Second)
	client.Close()

}
