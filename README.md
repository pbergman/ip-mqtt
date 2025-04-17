# IP MQTT

Was looking for an easy way to monitor my WAN IP on an [EdgeRouter 6P](https://eu.store.ui.com/eu/en/products/er-6p) to automatically change records upon changes. 

The router uses a mips64 architecture, and till this day golang does not support [plugins,](https://github.com/golang/go/issues/21222) so I choose to use mqtt to publish the ip changes.

This was the easiest solution because I had already and mqtt server running and had as side effect that multiple servers could listen and the router itself was only responsible to push the changes.     

It uses [netlink - linux routing sockets](netlink.go) to listen for realtime ip changes and [eclipse-paho/paho.golang](https://github.com/eclipse-paho/paho.golang) to publish to the mqtt server.

## Config

```ini
### mqtt server in URL form (tcp://username:password@host:port)
###
## required
server = ""

### interface to listen on, this can be mutiple interfaces
### by defining interface multiple times like
###   interface = wlp1s0
###   interface = etc0
### will make this listen on wlp1s0 and etc0 for changes
###
## required
interface = ""

### set concurrent workers for handling messages
# workers = 2

### set debugin to true for more verbose outpout
# debug = false

### toppic for pulbishing ip info
# mqtt_topic = "system/{{ hostname }}/networking/{{ interface }}/{{ protocol(ipv4|ipv6) }}"
```

## Building

To create a debian package for mips64 you can use to create a deb package 

```
make build-dev
```

## Installing

Copy the build file to your router 

```
scp build/ip-mqtt.mips64.deb user@192.168.1.1:~/
```

Install package

```
sudo dpkg -i ip-mqtt.mips64.deb
```

Change config

```
vi /etc/ip-mqtt.conf
```

Start service

```
sudo systemctl daemon-reload
sudo systemctl enable ip-mqtt
sudo systemctl start ip-mqtt
```

## Logs

Journal logs a disabled by default but can be set on with following config  

```
~:# show system systemd journal
 max-retention 60
 runtime-max-use 32
 storage volatile
```

And now you should see logs with

```
journalctl -u ip-mqtt -f
```
