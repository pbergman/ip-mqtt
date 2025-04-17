package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"syscall"
	
	"golang.org/x/sys/unix"
)

func handle(ctx context.Context, group *sync.WaitGroup, queue chan syscall.NetlinkMessage, app *App) {
	defer group.Done()

	for {
		select {
		case <-ctx.Done():

			if err := ctx.Err(); err != context.Canceled {
				app.GetLogger().Error(err)
			}

			return
		case message := <-queue:
			switch message.Header.Type {
			case unix.NLMSG_ERROR:
				app.GetLogger().Error(unix.Errno(-app.endianness.Uint32(message.Data[0:4])))
			case unix.RTM_DELADDR, unix.RTM_NEWADDR:

				var ifaddrmsg = getIfAddrMsg(message.Data)

				for _, name := range app.config.Interface {

					link, err := net.InterfaceByName(name)

					if err != nil {
						app.GetLogger().Error(fmt.Sprintf("could not find interface for %s: %s", name, err.Error()))
						continue
					}

					if ifaddrmsg.Index == uint32(link.Index) {

						attr, err := syscall.ParseNetlinkRouteAttr(&message)

						if err != nil {
							app.GetLogger().Error(fmt.Sprintf("could not decode netlink package: %s", err.Error()))
							continue
						}

						var ip net.IP

						if unix.RTM_DELADDR == message.Header.Type {
							switch ifaddrmsg.Family {
							case unix.AF_INET:
								ip = make(net.IP, net.IPv4len)
							case unix.AF_INET6:
								ip = make(net.IP, net.IPv6len)
							default:
								ip = make(net.IP, 0)
							}
						} else {
							if ipnet := getIpNet(attr, ifaddrmsg); ipnet != nil {
								ip = ipnet.IP
							}
						}

						var proto string

						if ifaddrmsg.Family == unix.AF_INET {
							proto = "ipv4"
						} else {
							proto = "ipv6"
						}

						app.GetLogger().Debug(fmt.Sprintf("[%s] %s: %s", name, proto, ip))

						if err := publish(app, ip, name, proto, app.topic); err != nil {
							app.GetLogger().Error(err.Error())
						}
					}
				}
			}
		}
	}
}
