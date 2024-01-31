FROM golang:1.20.4 as builder
WORKDIR /app
ENV GOMODULE="github.com/zhaoqiang0201/node_exporter" VERSION="v0.2.1"

COPY . /app/node_exporter
RUN go env -w GOPROXY="https://goproxy.cn,direct" && cd /app/node_exporter &&  go mod tidy && go build -ldflags="-X github.com/zhaoqiang0201/node_exporter/version.Version=v0.2.1" node_exporter.go

FROM ubuntu:18.04
WORKDIR /app
COPY --from=builder /app/node_exporter/node_exporter /app
COPY ./Shanghai /etc/localtime
ENTRYPOINT ["/app/node_exporter"]
CMD ["--path.procfs=/machine/proc","--path.sysfs=/machine/sys"]