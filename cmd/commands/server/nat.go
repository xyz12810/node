package server

import (
	"fmt"
	"github.com/ccding/go-stun/stun"
	"github.com/prestonTao/upnp"
	"net"
)

func StunPunch(preferedPort int) (string, int, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: preferedPort})
	defer conn.Close()
	if err != nil {
		return "", 0, err
	}

	client := stun.NewClientWithConnection(conn)
	client.SetVerbose(true)
	natType, host, err := client.Discover()
	if err != nil {
		return "", 0, err
	}
	fmt.Println("Discovered NAT type: ", natType.String())
	fmt.Println("Discovered IP info:", host.String())
	/*
		go func() {
			for {
				host, err := client.Keepalive()
				if err != nil {
					fmt.Println("Catched error on keepalive: ", err.Error())
					return
				}
				fmt.Println("Keepalive info: ", host.String())
				<-time.After(5 * time.Second)
			}
		}()
	*/
	return host.IP(), int(host.Port()), nil
}

func UPnpPunch(preferedPort int) (string, int, error) {
	service := &upnp.Upnp{}
	err := service.SearchGateway()
	if err != nil {
		return "", 0, err
	}

	err = service.ExternalIPAddr()
	if err != nil {
		return "", 0, err
	}

	fmt.Println("Local ip address: ", service.LocalHost)
	fmt.Println("Gateway: ", service.Gateway.Host)
	fmt.Println("Exteranl ip address: ", service.GatewayOutsideIP)

	err = service.AddPortMapping(preferedPort, preferedPort, "UDP")
	if err != nil {
		return "", 0, nil
	}
	return service.GatewayOutsideIP, preferedPort, nil
}
