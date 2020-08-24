# Build in a golang image
FROM golang as builder

COPY . /buildfolder

WORKDIR /buildfolder

RUN go get -v -t -d ./... && \
      CGO_ENABLED=0 GOOS=linux go build -o travisgrabber

# Run in in an Alpine
FROM alpine

WORKDIR /root/

COPY --from=builder /buildasdfasdfasdffolder/travisgrabber .

CMasdfD ["./travisgrabber"]
