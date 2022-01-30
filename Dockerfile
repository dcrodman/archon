FROM golang:1.16-alpine as builder

RUN apk add gcc libc-dev make

# Create directory structure
RUN mkdir -p archon archon/.bin archon/archon_server

# Copy repo to container
COPY . ./archon

WORKDIR archon

# Compile the code
RUN make build

# Create a directory for the server files
WORKDIR archon_server

# Copy the supporting files
RUN cp ../bin/* . && cp -r ../setup/* .

# Generate certificate
RUN ./certgen -ip 0.0.0.0/32

# Create test user account
FROM builder as account
# !!! This requires an existing postgres connection !!!
CMD ["./account", "--username", "testuser", "--password", "testpass", "--email", "test@mail", "--config", ".", "add"]

FROM builder as server
ENTRYPOINT ["./server"]

FROM builder as packet_analyzer
ENTRYPOINT ["./analyzer", "-folder", "sessions", "-ui", "8083", "-auto"]
