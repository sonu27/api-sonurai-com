FROM golang:1-alpine as builder
RUN apk add --no-cache git curl openssh ca-certificates && \
    curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

ENV PROJECT_DIR /app

COPY . $PROJECT_DIR
WORKDIR $PROJECT_DIR

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ./cmd/app/main ./cmd/app

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /app/cmd/app/main /app/
WORKDIR /app
CMD ["/app/main"]
