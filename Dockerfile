# syntax=docker/dockerfile:1
# Multi-stage build → distroless image with the PDFium-enabled binary
# (~23 MB on disk; pure-Go, no cgo).
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -tags pdfium -ldflags='-s -w' -o /distill ./cmd/distill

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /distill /usr/local/bin/distill
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/distill"]
CMD ["--help"]
