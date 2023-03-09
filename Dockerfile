FROM golang:latest as builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build

FROM alpine:3.17

WORKDIR /app

COPY --from=builder /app/linode-cli-autodeploy .

CMD ["/app/linode-cli-autodeploy", "serve"]