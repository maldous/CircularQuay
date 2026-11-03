# syntax=docker/dockerfile:1
FROM scratch
ARG TARGETARCH
COPY dist/cquay_linux_${TARGETARCH}/cquay /usr/local/bin/cquay
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/cquay"]

# Minimal offline CLI image. It intentionally contains no credentials, SOPS,
# Git, CA store or shell. Network/SOPS features require the separately verified
# tooling image/dependencies and explicit read-only mounts. Build metadata must
# pin base/tool artifacts and attach provenance before publication.
