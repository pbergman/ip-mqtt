package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func init() {
	flag.String("config", "/etc/ip-mqtt.conf", "config file")
	flag.Bool("print-interface", false, "print available network interface")
	flag.Bool("version", false, "print build version")
}

var (
	version string
)

func main() {

	flag.Parse()

	if flag.Lookup("version").Value.(flag.Getter).Get().(bool) {
		_, _ = fmt.Fprintf(os.Stdout, "version: %s\n", version)
		os.Exit(0)
	}

	if flag.Lookup("print-interface").Value.(flag.Getter).Get().(bool) {
		writerInterfaces()
		os.Exit(0)
	}

	app, err := bootstrap(flag.Lookup("config").Value.String())

	if err != nil {

		if nil != app {
			app.GetLogger().Error(err)
		}

		os.Exit(1)
	}

	listener, err := ListenNetlink()

	if nil != err {
		app.GetLogger().Error(err)
		os.Exit(1)
	}

	defer listener.Close()

	var queue = make(chan syscall.NetlinkMessage, 8)
	var fdSet = new(unix.FdSet)
	var tval = unix.NsecToTimeval((500 * time.Millisecond).Nanoseconds())
	var wg sync.WaitGroup

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	for i := 0; i < app.config.Workers; i++ {
		wg.Add(1)
		go handle(ctx, &wg, queue, app)
	}

outerLoop:

	for {

		select {

		case <-ctx.Done():
			break outerLoop

		default:
			ok, err := listener.Wait(fdSet, &tval)

			if err != nil {

				if v, ok := err.(unix.Errno); ok && v.Temporary() {
					continue
				}

				app.GetLogger().Error(err)
				stop()
				continue
			}

			if ok {
				messages, err := listener.ReadMessages()

				if err != nil {
					app.GetLogger().Error(fmt.Sprintf("failed reading netlink messages: %s", err))
					continue
				}

				for _, message := range messages {
					switch message.Header.Type {
					case unix.NLMSG_ERROR:
						app.GetLogger().Error(unix.Errno(-app.endianness.Uint32(message.Data[0:4])))
					case unix.RTM_DELADDR, unix.RTM_NEWADDR:
						queue <- message
					case unix.NLMSG_DONE:
						break
					}
				}
			}
		}
	}

	close(queue)
	wg.Wait()
}
