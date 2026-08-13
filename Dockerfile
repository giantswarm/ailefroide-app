FROM gsoci.azurecr.io/giantswarm/alpine:3.24.1

# architect/go-build produces one binary per target platform, so the multi-arch
# buildx build has to pick the one matching the stage it is building.
ARG TARGETARCH

RUN apk --no-cache add ca-certificates tzdata

COPY --chmod=0755 ./ailefroide-linux-${TARGETARCH} /opt/ailefroide
