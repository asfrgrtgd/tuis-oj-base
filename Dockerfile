# Pinned criyle/go-judge image (built 2025-11-27 UTC)
FROM criyle/go-judge@sha256:efea48ef7634efeec66f8ca3f04d9518212a1acc70cbc00e49d94f66df4bd8a9 AS upstream

# Install commonly used toolchains needed for typical online judge workloads.
# Pinned debian base (built 2025-11-17 UTC)
FROM debian@sha256:7cb087f19bcc175b96fbe4c2aef42ed00733a659581a80f6ebccfd8fe3185a3d
ARG GCC_VERSION=12
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        gcc-${GCC_VERSION} \
        g++-${GCC_VERSION} \
        libc6-dev \
        make \
        python3 python3-pip \
        openjdk-21-jdk-headless \
    && update-alternatives --install /usr/bin/gcc gcc /usr/bin/gcc-${GCC_VERSION} 100 \
    && update-alternatives --set gcc /usr/bin/gcc-${GCC_VERSION} \
    && update-alternatives --install /usr/bin/g++ g++ /usr/bin/g++-${GCC_VERSION} 100 \
    && update-alternatives --set g++ /usr/bin/g++-${GCC_VERSION} \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /opt
COPY --from=upstream /opt/go-judge /opt/mount.yaml /opt/

EXPOSE 5050 5051 5052
ENTRYPOINT ["./go-judge"]
