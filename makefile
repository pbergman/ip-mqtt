.ONESHELL:
DOCKER_IMAGE_NAME := debian-build-env
DOCKER_BUILD_ARCH := mips64

docker-stop:
	sudo docker stop $(DOCKER_IMAGE_NAME)-$(DOCKER_BUILD_ARCH)

docker-start: docker-build
	sudo docker run \
		--rm \
		--detach \
		--tty \
		--interactive \
		--env GOOS=linux \
		--env GOARCH=$(DOCKER_BUILD_ARCH) \
		--env CGO_ENABLED=1 \
		--env CC=mips64-linux-gnuabi64-gcc \
		--mount type=bind,source="$$(pwd)",target=/app \
		--workdir /app \
		--entrypoint bash \
		--name $(DOCKER_IMAGE_NAME)-$(DOCKER_BUILD_ARCH) \
	  	$(DOCKER_IMAGE_NAME):$(DOCKER_BUILD_ARCH)

docker-build:
	sudo docker build --tag $(DOCKER_IMAGE_NAME):$(DOCKER_BUILD_ARCH) .

build-go:
	go build  -o ./build/ip-mqtt -ldflags '-s -w -extldflags "-static" -X main.version=$(shell cat .version)' && mips64-linux-gnuabi64-strip ./build/ip-mqtt

build-mips64le:
	mkdir -p "./build"
	sudo docker exec -it $(DOCKER_IMAGE_NAME)-$(DOCKER_BUILD_ARCH) make build-go
	sudo chown $$(id -u):$$(id -g) -R ./build/

set_version:
	echo "$(shell git describe --tags --always --long)" > .version

build: set_version docker-start build-mips64le docker-stop

write-deb-dirs:
	mkdir -p ./build/src/DEBIAN
	mkdir -p ./build/src/usr/share/ip-mqtt
	mkdir -p ./build/src/usr/local/bin

write-deb-postinst:
	mkdir -p ./build/src/DEBIAN
	cat > ./build/src/DEBIAN/postinst <<-'EOF'
		#!/bin/bash
		set -e
		if [ ! -f '/etc/systemd/system/multi-user.target.wants/ip-mqtt.service' ]; then
			# assume when running we don`t need to check config
			if [ ! -f '/etc/ip-mqtt.conf' ]; then
				cp /usr/share/ip-mqtt/ip-mqtt.conf /etc/ip-mqtt.conf
				chmod 600 /etc/ip-mqtt.conf
			fi
			if [ ! -f '/etc/systemd/system/ip-mqtt.service' ]; then
				cp /usr/share/ip-mqtt/ip-mqtt.service /etc/systemd/system/ip-mqtt.service
				chmod 755 /etc/systemd/system/ip-mqtt.service
			fi
			systemctl daemon-reload
			systemctl enable ip-mqtt.service
			echo "Please configure your settings in /etc/ip-mqtt.conf and then start with \"sudo systemctl start ip-mqtt\""
		fi
	EOF
	sudo chmod 755 ./build/src/DEBIAN/postinst

write-deb-prerm:
	cat > ./build/src/DEBIAN/prerm <<-'EOF'
		#!/bin/bash
		set -e
		if [[ "$$1" = "remove" ]]; then
			if [ -f '/etc/systemd/system/multi-user.target.wants/ip-mqtt.service' ]; then
				systemctl --no-reload disable ip-mqtt.service
				systemctl stop ip-mqtt.service &> /dev/null
			fi
			if [ -f "/etc/ip-mqtt.conf" ]; then
				rm /etc/ip-mqtt.conf
			fi
			if [ -f /etc/systemd/system/ip-mqtt.service ]; then
				rm /etc/systemd/system/ip-mqtt.service
			fi
		fi
	EOF
	sudo chmod 755 ./build/src/DEBIAN/prerm

write-deb-control:
	mkdir -p ./build/src/DEBIAN
	cat > ./build/src/DEBIAN/control <<EOF
		Package: ip-mqtt
		Version: $(shell cat .version)
		Maintainer: Philip Bergman <pbergman@live.nl>
		Architecture: mips
		Description: network interface listener that will publish ip changes to mqtt server
	EOF

write-deb-systemtd:
	cat > ./build/src/usr/share/ip-mqtt/ip-mqtt.service <<EOF
		[Unit]
		Description=network interface listener that will publish ip changes to mqtt server
		[Service]
		User=root
		ExecStart=/usr/local/bin/ip-mqtt
		Restart=always
		RestartSec=2
		[Install]
		WantedBy=multi-user.target
	EOF
	sudo chmod 755 ./build/src/usr/share/ip-mqtt/ip-mqtt.service

build-deb: build write-deb-dirs write-deb-control write-deb-systemtd write-deb-postinst write-deb-prerm
	set -e
	mkdir -p ./build/src/usr/share/ip-mqtt
	cp ./config_example.conf ./build/src/usr/share/ip-mqtt/ip-mqtt.conf
	cp ./build/ip-mqtt ./build/src/usr/local/bin/ip-mqtt
	sudo chown root:root -R ./build/src
	dpkg-deb --build ./build/src ./build/ip-mqtt.$(DOCKER_BUILD_ARCH).$(shell cat .version).deb
	sudo rm ./build/src -rf
	sudo rm ./build/ip-mqtt

