FROM gsoci.azurecr.io/giantswarm/alpine:3.24.1

# architect/go-build produces one binary per target platform, so the multi-arch
# buildx build has to pick the one matching the stage it is building.
ARG TARGETARCH

RUN apk --no-cache add ca-certificates tzdata

# The binary is named after the repository. devctl derives go-build's `binary`
# param from the repo name and offers no override, so this must stay
# ailefroide-app-* even though the image and the installed path are
# ailefroide. See giantswarm/github#5775.
COPY --chmod=0755 ./ailefroide-app-linux-${TARGETARCH} /opt/ailefroide
