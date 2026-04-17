FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /handler ./lambda/api

FROM public.ecr.aws/lambda/provided:al2023-arm64
COPY --from=builder /handler /handler
ENTRYPOINT ["/handler"]
