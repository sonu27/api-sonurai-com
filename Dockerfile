FROM golang:1.20-alpine as builder

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY ./ ./

RUN go build -trimpath -mod=readonly -o /app ./cmd/app

FROM gcr.io/distroless/static-debian11

USER nonroot:nonroot

COPY --from=builder --chown=nonroot:nonroot /app /app

CMD ["/app"]
