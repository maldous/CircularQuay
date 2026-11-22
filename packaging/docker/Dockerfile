# syntax=docker/dockerfile:1
FROM scratch
ARG TARGETARCH
COPY dist/cq_linux_${TARGETARCH}/cq /usr/local/bin/cq
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/cq"]

# Minimal offline CLI image. It intentionally contains no credentials, SOPS,
# Git, CA store or shell. Network/SOPS features require the separately verified
# tooling image/dependencies and explicit read-only mounts. Build metadata must
# pin base/tool artifacts and attach provenance before publication.
