# Build the terrascaler binary.
FROM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH
ARG LDFLAGS

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -ldflags="${LDFLAGS}" -o terrascaler ./cmd/terrascaler

# Use distroless as minimal base image to package the binary.
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/terrascaler .
USER 65532:65532

ENTRYPOINT ["/terrascaler"]
