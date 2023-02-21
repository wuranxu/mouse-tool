package mouse_tool

import (
	"errors"
	"net"
)

var (
	ErrorGetExternalIP = errors.New("get external ip error, are you connected to network")
)

func GetExternalIP() (ip string, err error) {
	var ifaces []net.Interface
	ifaces, err = net.Interfaces()
	if err != nil {
		return
	}
	for _, i := range ifaces {
		var addrs []net.Addr
		addrs, err = i.Addrs()
		if err != nil {
			return
		}
		for _, addr := range addrs {
			var netIp net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				netIp = v.IP
			case *net.IPAddr:
				netIp = v.IP
			}
			if netIp == nil || netIp.IsLoopback() {
				continue
			}
			ipv4 := netIp.To4()
			if ipv4 == nil {
				continue
			}
			ip = ipv4.String()
			return
		}
	}
	err = ErrorGetExternalIP
	return
}
