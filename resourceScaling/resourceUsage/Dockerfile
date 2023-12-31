#编译镜像
FROM golang:1.15 as builder
MAINTAINER bit_scg "uzz_scg@163.com"
ENV GOPROXY https://goproxy.cn
ENV GO111MODULE on
WORKDIR /usr/local/go/src/resourceUsage
ADD ./go.mod .
ADD ./go.sum .
RUN go mod download
ADD .  /usr/local/go/src/resourceUsage/
WORKDIR /usr/local/go/src/resourceUsage

#go构建可执行文件,-o 生成Server，放在当前目录
RUN go build -ldflags="-w -s" -o resourceUsage .

#执行镜像
FROM ubuntu:latest
WORKDIR /root/go/resourceUsage
COPY --from=builder /usr/local/go/src/resourceUsage .
#RUN chmod +x ./resourceUsage
ENTRYPOINT  ["./resourceUsage"]
