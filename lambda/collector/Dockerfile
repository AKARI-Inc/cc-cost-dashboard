FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOARCH=arm64 go build -o /handler ./lambda/collector

FROM public.ecr.aws/lambda/provided:al2023-arm64
COPY --from=builder /handler /handler
ENTRYPOINT ["/handler"]
