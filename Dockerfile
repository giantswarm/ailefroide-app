FROM quay.io/giantswarm/alpine:3.12
RUN apk --no-cache add ca-certificates
RUN mkdir -p /opt
COPY ./ailefroide /opt/ailefroide
RUN chmod +x /opt/ailefroide
