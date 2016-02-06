package peer

import (
	"fmt"
	"net"
	"time"

	"github.com/sdcoffey/olympus/fs"
)

const (
	address = "224.0.0.1:5353"
	maxSize = fs.KILOBYTE
)

func FindServer(timeout time.Duration) (net.IP, error) {
	if addr, err := net.ResolveUDPAddr("udp4", address); err != nil {
		return net.IPv4zero, err
	} else if socket, err := net.ListenMulticastUDP("udp4", nil, addr); err != nil {
		return net.IPv4zero, err
	} else {
		defer socket.Close()
		socket.SetReadDeadline(time.Now().Add(timeout))

		for {
			socket.SetReadBuffer(maxSize)
			b := make([]byte, maxSize)
			if _, src, err := socket.ReadFromUDP(b); err != nil {
				return net.IPv4zero, err
			} else {
				return src.IP, nil
			}
		}
	}
}

func ClientHeartbeat() {
	if addr, err := net.ResolveUDPAddr("udp4", address); err != nil {
		fmt.Println(err.Error())
	} else if connection, err := net.DialUDP("udp4", nil, addr); err != nil {
		fmt.Println(err.Error())
	} else {
		ticker := time.Tick(time.Second) // todo config
		for {
			<-ticker
			if _, err := connection.Write([]byte("HELLO")); err != nil {
				fmt.Println(err.Error())
			}
		}
	}
}
