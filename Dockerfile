FROM golang:1.15-alpine as builder

RUN apk add --no-cache \
    alpine-sdk \
    ca-certificates \
    tzdata

# Force the go compiler to use modules
ENV GO111MODULE=on
# Create the user and group files to run unprivileged 
RUN mkdir /user && \
    echo 'nobody:x:65534:65534:nobody:/:' > /user/passwd && \
    echo 'nobody:x:65534:' > /user/group

ADD . /app
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux go build -a -o rapg .

FROM scratch

# copy files from other container
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /user/group /user/passwd /etc/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/rapg /rapg

USER nobody:nobody
ENTRYPOINT ["/rapg"]

EXPOSE 8787