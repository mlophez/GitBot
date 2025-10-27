FROM golang:1.22.2-alpine as builder
RUN apk --no-cache add ca-certificates
WORKDIR /src
COPY . .
RUN cd ./cmd/server && GOOS=linux go build -o /gitbot

FROM scratch
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder --chmod=755 /gitbot /app/gitbot
COPY --from=builder --chmod=755 /src/env.local.ini /app/env.ini
ENTRYPOINT ["/app/gitbot"]
