package main

import (
	"fmt"
	"net"
	"os"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func ListenNetlink() (*NetlinkListener, error) {

	/** https://man7.org/linux/man-pages/man7/rtnetlink.7.html */
	socket, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_DGRAM, unix.NETLINK_ROUTE)

	if err != nil {
		return nil, fmt.Errorf("socket: %s", err)
	}

	var addr = &unix.SockaddrNetlink{
		Family: unix.AF_NETLINK,
		Pid:    uint32(0),
		Groups: uint32((1 << (unix.RTNLGRP_LINK - 1)) | (1 << (unix.RTNLGRP_IPV4_IFADDR - 1)) | (1 << (unix.RTNLGRP_IPV6_IFADDR - 1))),
	}

	if err := unix.Bind(socket, addr); err != nil {
		return nil, fmt.Errorf("bind: %s", err)
	}

	var pool = &sync.Pool{
		New: func() any {
			return make([]byte, os.Getpagesize())
		},
	}

	return &NetlinkListener{fd: socket, addr: addr, pool: pool}, nil
}

type NetlinkListener struct {
	fd   int
	addr *unix.SockaddrNetlink
	pool *sync.Pool
}

func (l *NetlinkListener) Close() error {
	return unix.Close(l.fd)
}

func (l *NetlinkListener) Wait(fdSet *unix.FdSet, timeout *unix.Timeval) (bool, error) {

	if fdSet == nil {
		fdSet = new(unix.FdSet)
	}

	fdSet.Set(l.fd)

	n, err := unix.Select(l.fd+1, fdSet, nil, nil, timeout)

	if err != nil {
		return false, err
	}

	if n > 0 && fdSet.IsSet(l.fd) {
		return true, nil
	}

	return false, nil

}

func (l *NetlinkListener) ReadMessages() (msgs []syscall.NetlinkMessage, err error) {
	defer func() {
		if rec := recover(); rec != nil {
			if v, ok := rec.(error); ok {
				err = v
			}
		}
	}()

	var buf = l.pool.Get().([]byte)

	defer l.pool.Put(buf)

	n, err := unix.Read(l.fd, buf)

	if err != nil {
		return nil, fmt.Errorf("read: %s", err)
	}

	if n < unix.NLMSG_HDRLEN {
		return nil, fmt.Errorf("short response from netlink (%d)", n)
	}

	msgs, err = syscall.ParseNetlinkMessage(buf[:n])

	if err != nil {
		return nil, fmt.Errorf("parse: %s", err)
	}

	return msgs, nil
}

func getIfAddrMsg(data []byte) *unix.IfAddrmsg {
	return (*unix.IfAddrmsg)(unsafe.Pointer(&data[0:unix.SizeofIfAddrmsg][0]))
}

func getIpNet(routeAttr []syscall.NetlinkRouteAttr, addrMsg *unix.IfAddrmsg) *net.IPNet {
	var dst *net.IPNet
	var loc *net.IPNet

	for _, attribute := range routeAttr {
		switch attribute.Attr.Type {
		case unix.IFA_ADDRESS:
			dst = &net.IPNet{
				IP:   attribute.Value,
				Mask: net.CIDRMask(int(addrMsg.Prefixlen), 8*len(attribute.Value)),
			}
		case unix.IFA_LOCAL:
			var n = 8 * len(attribute.Value)
			loc = &net.IPNet{
				IP:   attribute.Value,
				Mask: net.CIDRMask(n, n),
			}
		}
	}

	if loc != nil && (addrMsg.Family != unix.AF_INET || (nil != dst && false == loc.IP.Equal(dst.IP))) {
		return loc
	}

	return dst
}
