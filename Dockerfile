# syntax = docker/dockerfile:1
FROM golang:1.25-alpine as builder

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY ./ ./

RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -buildvcs=false -mod=readonly -o /app ./cmd/app

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder --chown=nonroot:nonroot /app /app

CMD ["/app"]
