# IP MQTT

Was looking for an easy way to monitor my WAN ip on an [EdgeRouter 6P](https://eu.store.ui.com/eu/en/products/er-6p) to update dns records on changes. 

My first approach was to creat an application which uses plugins for the different providers but golang does not supporting plugins on [linux/mips](https://github.com/golang/go/issues/21222). 

Because of this I opted to use mqtt to publish the ip addresses and create listeners which can listen to the topics for changes.   

It uses netlink - [linux routing sockets](netlink.go) to listen for realtime ip changes.   

## Building

To create a debian package for mips64 which you can use to install on the router. 

```
make build-dev
```

## Installing

Copy the build file tou router 

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
