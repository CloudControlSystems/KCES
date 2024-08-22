#编译镜像
FROM golang:1.15 as builder
MAINTAINER bit_scg "uzz_scg@163.com"

ENV GO111MODULE on
ENV GOPROXY https://goproxy.cn,direct
COPY . /usr/local/go/src/taskContainerBuilder

WORKDIR /usr/local/go/src/taskContainerBuilder
RUN go mod download
#go构建可执行文件,-o 生成Server，放在当前目录
RUN GO111MODULE=on go build -ldflags="-w -s" -o taskContainerBuilder .

#执行镜像
FROM ubuntu:latest
WORKDIR /root/go/taskContainerBuilder
COPY --from=builder /usr/local/go/src/taskContainerBuilder .
ENTRYPOINT  ["./taskContainerBuilder"]