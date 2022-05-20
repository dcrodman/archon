Dockerfile
FROM golang:1.16-alpine as builder

RUN apk add gcc libc-dev make

# Copy repo to container
COPY . /app

# Compile the code
WORKDIR /app
RUN make build

FROM alpine:latest

COPY --from=builder /app/bin/ /usr/bin
# Copy the supporting files
COPY ../setup/ /etc/archon

CMD ["server", "--config", "/etc/archon"]