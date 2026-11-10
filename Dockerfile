FROM golang:1.27 AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 go build -o server ./cmd/api

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=builder /app/server /app/server
EXPOSE 8080
ENTRYPOINT ["/app/server"]
