FROM golang:1.23-alpine AS builder

WORKDIR /build

ADD go.mod .
ADD go.sum .
RUN go mod download

COPY cmd cmd
COPY internal internal
COPY pkg pkg


RUN GOOS=linux go build -o app ./cmd

FROM golang:1.23-alpine
WORKDIR /root/


COPY --from=builder /build/app .
ADD config/config.yaml ./config/config.yaml

EXPOSE 8080

CMD ["./app"]