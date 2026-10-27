# Minimal toolchain image for Taiji source builds (no Wuji CLI on PATH).
FROM fedora:42

RUN dnf install -y \
    git \
    golang \
    gcc-c++ \
    cmake \
    ncurses-devel \
    zlib-devel \
    openssl-devel \
    cargo \
    python3 \
    make \
    rsync \
    && dnf clean all

# Deliberately exclude user-local bins (no wuji / wuji-core CLI from a host install).
ENV PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
ENV CGO_ENABLED=1
ENV GOTOOLCHAIN=auto
ENV HOME=/root

WORKDIR /work
