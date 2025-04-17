FROM debian:bookworm-slim

RUN dpkg --add-architecture mips64el

RUN apt update && apt install -y \
    build-essential  \
    libc6-dev:mips64el \
    wget \
    gcc-mips64-linux-gnuabi64 \
    binutils-mips64-linux-gnuabi64

ENV GO_VERSION=1.23.5
RUN wget -qO- https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz | tar -xzC /usr/local
RUN ln -s /usr/local/go/bin/go /usr/bin/go


