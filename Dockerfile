FROM docker.io/library/golang:1.17 AS builder

#### Set Go environment
# Disable CGO to create a self-contained executable
# Do not enable unless it's strictly necessary
ENV CGO_ENABLED 0
# Set Linux as target
ENV GOOS linux

### Prepare base image
RUN apt-get update && apt-get install -y zip ca-certificates tzdata
WORKDIR /usr/share/zoneinfo
RUN zip -r -0 /zoneinfo.zip .
RUN useradd --home /app/ -M appuser

WORKDIR /src/

### Copy Go modules files and cache dependencies
# If dependencies do not changes, these two lines are cached (speed up the build)
COPY go.* ./
RUN go mod download

### Copy Go code
COPY cmd cmd
COPY service service

RUN go generate ./...

### Build executables, strip debug symbols
WORKDIR /src/cmd/
RUN mkdir /app/
RUN /bin/bash -c "for ex in \$(ls); do pushd \$ex; go build -mod=readonly -ldflags \"-extldflags \\\"-static\\\"\" -a -installsuffix cgo -o /app/\$ex .; popd; done"
RUN cd /app/ && strip *

### Create final container from scratch
FROM scratch

### Inform Docker about which port are used
EXPOSE 3001 3002

### Populate scratch with CA certificates and Timezone infos from the builder image
ENV ZONEINFO /zoneinfo.zip
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /zoneinfo.zip /
COPY --from=builder /etc/passwd /etc/passwd

### Copy the build executable from the builder image
WORKDIR /app/
COPY --from=builder /app/* ./

### Downgrade to user level (from root)
USER appuser

### Executable command
CMD ["/app/http-telescope"]
