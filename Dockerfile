FROM golang:1.10 as builder

# Cloud builder defaults to /workspace as workspace main directory. Additionally,
# specify the WORKDIR to be the location of our application source code which 
# should be inside of GOPATH using standard Golang directory patterns.
ENV GOPATH=/workspace
WORKDIR $GOPATH/src/github.com/joinhandshake/kubekite

# Download and install the latest release of dep, which we use for vendoring our
# dependencies.
ADD https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep

# Add just our Gopkg configurations to WORKDIR and install dependencies.
COPY Gopkg.toml Gopkg.lock ./
RUN dep ensure --vendor-only

# Lastly, add the rest of the code and build the binary.
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app ./src/kubekite

# Throw out the build step of the docker image and start fresh on Alpine Linux
FROM golang:1.10

# For our final image, we'll work out of /app directory. This
# is more flexible, our standard directory for Handshake services
# is /app.
WORKDIR /app/

COPY job-templates/job.yaml /

# Add the binary from our builder stage to the image and set the default CMD
COPY --from=builder /app /app/
CMD ["./app"]