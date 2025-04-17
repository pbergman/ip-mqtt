package main

import (
	"fmt"
	"net"
	"os"
	"text/tabwriter"
)

func validateInterfaces(app *App) error {

	if len(app.config.Interface) == 0 {
		x, err := getAllSystemInterfaces(app)

		if err != nil {
			return err
		}

		app.config.Interface = x

		return nil
	}

	for _, name := range app.config.Interface {
		if _, err := net.InterfaceByName(name); err != nil {
			return err
		}
	}

	return nil
}

func getAllSystemInterfaces(app *App) ([]string, error) {
	var list = make([]string, 0)

	interfaces, err := net.Interfaces()

	if err != nil {
		return nil, fmt.Errorf("could not get system interfaces: %s", err)
	}

	for _, i := range interfaces {
		app.GetLogger().Debug(fmt.Sprintf("using interface '%s' (%d)", i.Name, i.Index))
		list = append(list, i.Name)
	}

	return list, nil
}

func writerInterfaces() {
	intfs, err := net.Interfaces()

	if err != nil {
		_ = fmt.Errorf("failed to read interface: %s", err)
		return
	}

	var writer = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

	for _, intf := range intfs {

		if addrs, _ := intf.Addrs(); addrs != nil {
			for _, x := range addrs {
				if addr, ok := x.(*net.IPNet); ok {
					var ip string

					if ipv4 := addr.IP.To4(); ipv4 != nil && len(ipv4) == net.IPv4len {
						ip = ipv4.String()
					} else {
						ip = addr.IP.To16().String()
					}

					_, _ = fmt.Fprintf(writer, "%d\t%s\t%s\n", intf.Index, intf.Name, ip)
				}
			}
		}
	}

	_ = writer.Flush()
}

func registerInterfaces(app *App) error {

	for _, name := range app.config.Interface {

		link, err := net.InterfaceByName(name)

		if err != nil {
			return fmt.Errorf("net: %s", err)
		}

		addrs, err := link.Addrs()

		if err != nil {
			return fmt.Errorf("net.address: %s", err)
		}

		for _, x := range addrs {
			if addr, ok := x.(*net.IPNet); ok {

				var ip net.IP
				var proto string

				if ipv4 := addr.IP.To4(); ipv4 != nil && len(ipv4) == net.IPv4len {
					ip = ipv4
					proto = "ipv4"
				} else {
					ip = addr.IP.To16()
					proto = "ipv6"
				}

				app.GetLogger().Debug(fmt.Sprintf("[%s] %s: %s", name, proto, ip))

				if err := publish(app, ip, name, proto, app.topic); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
