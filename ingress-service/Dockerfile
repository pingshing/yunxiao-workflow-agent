FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN go build -o /out/ingress-service ./cmd/server

FROM alpine:3.21
WORKDIR /app
COPY --from=build /out/ingress-service /app/ingress-service
EXPOSE 8080
ENTRYPOINT ["/app/ingress-service"]
