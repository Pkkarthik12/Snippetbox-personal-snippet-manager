FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/snippetbox ./cmd/server

FROM alpine:3.20
WORKDIR /app
RUN adduser -D -H snippetbox
COPY --from=build /out/snippetbox /usr/local/bin/snippetbox
COPY templates ./templates
COPY static ./static
USER snippetbox
EXPOSE 8080
CMD ["snippetbox"]
