FROM golang:alpine as builder

RUN apk add gcc libc-dev
# Create initial dir
RUN mkdir archon
COPY ./ ./archon
WORKDIR archon
# Compile the code
RUN mkdir .bin
ENV GOBIN=/go/archon/.bin
RUN go install ./cmd/*
# Create a directory for the server files
RUN mkdir archon_server
WORKDIR archon_server
RUN cp ../.bin/* .
# Copy the supporting files
RUN cp -r ../setup/* .
# Add override config
COPY /build/override.yaml ./override.yaml
# Generate certificate
RUN ./generate_cert -ip 0.0.0.0/32
# Create test user account
FROM builder as account
# !!! This requires an existing postgres connection !!!
CMD ["./account", "-username", "testuser", "-password", "testpass", "-email", "test@mail", "add"]

FROM builder as server
ENTRYPOINT ["./server"]

FROM builder as packet_analyzer
ENTRYPOINT ["./packet_analyzer", "-capture", "-addr", "0.0.0.0", "-folder", "sessions", "-manage", "8083", "-auto"]
