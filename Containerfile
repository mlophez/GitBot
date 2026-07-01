FROM --platform=$BUILDPLATFORM golang:1.22.2-alpine as builder
RUN apk --no-cache add ca-certificates
# BuildKit/Podman inject these automatically per --platform target.
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY . .
# CGO stays off (scratch base, static binary); cross-compile for the requested target.
RUN cd ./cmd/server && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /gitbot
RUN cd ./cmd/repair && CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /reconcile

FROM scratch
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder --chmod=755 /gitbot /app/gitbot
COPY --from=builder --chmod=755 /reconcile /app/reconcile
COPY --from=builder --chmod=755 /src/env.local.ini /app/env.ini
ENTRYPOINT ["/app/gitbot"]
