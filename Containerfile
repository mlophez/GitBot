FROM golang:1.22.2-alpine as builder
WORKDIR /src
COPY . .
RUN cd ./cmd && go build -o /gitbot

FROM scratch
WORKDIR /app
COPY --from=builder --chmod=755 /gitbot /app/gitbot
COPY --from=builder --chmod=755 /src/env.local.ini /app/env.ini
ENTRYPOINT ["/app/gitbot"]
