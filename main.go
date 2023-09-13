package main

import (
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"net"
	"syscall"
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
	version, err := GetKernelVersion()
	if err != nil {
		panic(fmt.Errorf("failed to get kernel version: %s", err.Error()))
	}
	if version.Kernel < 4 || (version.Kernel == 4 && version.Major < 18) {
		panic(fmt.Errorf("UDP_SEGMENT option is not supported for linux %s, use 4.18 or higher", version.String()))
	}
	if err := rawConn.Control(func(fd uintptr) {
		ret, serr := unix.GetsockoptInt(int(fd), unix.IPPROTO_UDP, unix.UDP_SEGMENT)
		if ret == -1 {
			panic(fmt.Errorf("GSO is not supported: sockopt is -1: %s", serr.Error()))
		}
		if serr != nil {
			if errno, ok := serr.(syscall.Errno); ok {
				panic(fmt.Errorf("GSO is not supported: errno is %d: %s", int(errno), serr.Error()))

			} else {
				panic(fmt.Errorf("GSO is not supported: %s", serr.Error()))
			}
		}
		fmt.Printf("GSO is supported: sockopt is %d\n", ret)
	}); err != nil {
		panic(fmt.Errorf("GSO is not supported: %s", err.Error()))
	}

}

// GetKernelVersion gets the current kernel version.
// source: https://github.com/moby/moby/blob/v24.0.6/pkg/parsers/kernel/kernel_unix.go
func GetKernelVersion() (*VersionInfo, error) {
	uts, err := uname()
	if err != nil {
		return nil, err
	}

	// Remove the \x00 from the release for Atoi to parse correctly
	return ParseRelease(unix.ByteSliceToString(uts.Release[:]))
}

// VersionInfo holds information about the kernel.
// source: https://github.com/moby/moby/blob/v24.0.6/pkg/parsers/kernel/kernel.go
type VersionInfo struct {
	Kernel int    // Version of the kernel (e.g. 4.1.2-generic -> 4)
	Major  int    // Major part of the kernel version (e.g. 4.1.2-generic -> 1)
	Minor  int    // Minor part of the kernel version (e.g. 4.1.2-generic -> 2)
	Flavor string // Flavor of the kernel version (e.g. 4.1.2-generic -> generic)
}

// source: https://github.com/moby/moby/blob/v24.0.6/pkg/parsers/kernel/kernel.go
func (k *VersionInfo) String() string {
	return fmt.Sprintf("%d.%d.%d%s", k.Kernel, k.Major, k.Minor, k.Flavor)
}

// ParseRelease parses a string and creates a VersionInfo based on it.
// source: https://github.com/moby/moby/blob/v24.0.6/pkg/parsers/kernel/kernel.go
func ParseRelease(release string) (*VersionInfo, error) {
	var (
		kernel, major, minor, parsed int
		flavor, partial              string
	)

	// Ignore error from Sscanf to allow an empty flavor.  Instead, just
	// make sure we got all the version numbers.
	parsed, _ = fmt.Sscanf(release, "%d.%d%s", &kernel, &major, &partial)
	if parsed < 2 {
		return nil, errors.New("Can't parse kernel version " + release)
	}

	// sometimes we have 3.12.25-gentoo, but sometimes we just have 3.12-1-amd64
	parsed, _ = fmt.Sscanf(partial, ".%d%s", &minor, &flavor)
	if parsed < 1 {
		flavor = partial
	}

	return &VersionInfo{
		Kernel: kernel,
		Major:  major,
		Minor:  minor,
		Flavor: flavor,
	}, nil
}

// source: https://github.com/moby/moby/blob/v24.0.6/pkg/parsers/kernel/uname_linux.go
func uname() (*unix.Utsname, error) {
	uts := &unix.Utsname{}

	if err := unix.Uname(uts); err != nil {
		return nil, err
	}
	return uts, nil
}
