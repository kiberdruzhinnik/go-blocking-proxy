FROM golang:alpine AS builder
WORKDIR /src/app
COPY go.mod ./
COPY . ./
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o go-blocking-proxy .

FROM gcr.io/distroless/base-debian13:nonroot
WORKDIR /app
COPY --from=builder /src/app/go-blocking-proxy ./go-blocking-proxy
EXPOSE 1080
ENTRYPOINT ["./go-blocking-proxy", "-listen", ":1080"]
