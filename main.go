package main

import (
	"fmt"
	"golang.org/x/sys/unix"
	"net"
)

func main() {
	addr, err := net.ResolveUDPAddr("udp", "localhost:0")
	if err != nil {
		panic(err)
	}
	udpConn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}
	rawConn, err := udpConn.SyscallConn()
	if err != nil {
		panic(err)
	}
	var serr error
	if err := rawConn.Control(func(fd uintptr) {
		_, serr = unix.GetsockoptInt(int(fd), unix.IPPROTO_UDP, unix.UDP_SEGMENT)
	}); err != nil {
		panic(fmt.Errorf("GSO is not supported: %s", err.Error()))
	}
	if serr != nil {
		panic(fmt.Errorf("GSO is not supported: %s", serr.Error()))
	}
	fmt.Printf("GSO is supported\n")
}
