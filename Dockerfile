FROM gsoci.azurecr.io/giantswarm/alpine:3.16.2
RUN apk --no-cache add ca-certificates tzdata
RUN mkdir -p /opt
COPY ./ailefroide /opt/ailefroide
RUN chmod +x /opt/ailefroide
