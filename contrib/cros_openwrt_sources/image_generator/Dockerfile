# Dockerfile to set up an OpenWrt build environment for MediaTek Filogic
# Base Image: Ubuntu 22.04 LTS
FROM ubuntu:22.04

# OPENWRT_COMMIT sha of commit from which image would be build
ARG OPENWRT_COMMIT=""
# MTK_OPENWRT_FEEDS_COMMIT used for BPi R4 build only, could be empty for other devices
ARG MTK_OPENWRT_FEEDS_COMMIT=""
# DEVICE is an arg that used to choose a folder were configs stored. It should match one of a folder name in ./configs
ARG DEVICE=""

# Validate required args to not be empty
RUN if [ -z "$OPENWRT_COMMIT" ]; then echo "OPENWRT_COMMIT cannot be empty!" && exit 1; fi && \
    if [ -z "$DEVICE" ]; then echo "DEVICE cannot be empty!" && exit 1; fi

# Set environment variables to prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Set a default language to prevent locale errors
ENV LANG=C.UTF-8

# Install prerequisites for OpenWrt build system as specified by the user
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    build-essential \
    clang \
    flex \
    bison \
    g++ \
    gawk \
    gcc-multilib \
    g++-multilib \
    gettext \
    git \
    libncurses-dev \
    libssl-dev \
    python3-distutils \
    python3-setuptools \
    python-is-python3 \
    rsync \
    swig \
    unzip \
    zlib1g-dev \
    file \
    wget \
    ca-certificates \
    uuid-runtime \
    && rm -rf /var/lib/apt/lists/*

# Set up the main build directory
WORKDIR /build

# --- Source Code Checkout ---
# Get OpenWrt source code (specific branch and commit)
RUN git clone https://git.openwrt.org/openwrt/openwrt.git openwrt && \
    cd openwrt && \
    echo "Checking out OpenWrt commit ${OPENWRT_COMMIT}" && \
    git checkout ${OPENWRT_COMMIT} && \
    cd .. && \
    echo "OpenWrt checkout complete."

# Get mtk-openwrt-feeds source code (specific branch and commit)
RUN \
    if [ -n "${MTK_OPENWRT_FEEDS_COMMIT}" ]; then \
        echo "MTK_OPENWRT_FEEDS_COMMIT is set. Proceeding with MediaTek feeds clone and checkout." && \
        git clone https://git01.mediatek.com/openwrt/feeds/mtk-openwrt-feeds mtk-openwrt-feeds && \
        cd mtk-openwrt-feeds && \
        echo "Checking out mtk-openwrt-feeds commit ${MTK_OPENWRT_FEEDS_COMMIT}" && \
        git checkout "${MTK_OPENWRT_FEEDS_COMMIT}" && \
        cd .. && \
        echo "MediaTek feeds checkout complete."; \
    else \
        echo "MTK_OPENWRT_FEEDS_COMMIT environment variable is empty. Skipping MediaTek feeds clone and checkout."; \
    fi


# Change working directory to where OpenWrt source is for the autobuild script
WORKDIR /build/openwrt

VOLUME /build/openwrt/bin
VOLUME /build/openwrt/logs

COPY custom_files /build/openwrt/files
COPY custom_packages /build/openwrt/package
COPY configs /build/openwrt/configs
COPY run.sh /build/openwrt/

# Potentially needed by some configure scripts within the OpenWrt build,
# especially if run by a script that might have elevated (root) context perception.
ENV FORCE_UNSAFE_CONFIGURE=1

# Workaround to make perf lib and prevent error:
# Makefile.config:881: *** ERROR: No python interpreter needed for jevents generation. Install python or build with NO_JEVENTS=1.. Stop.
# make[4]: *** [Makefile.perf:242: sub-make] Error 2
ENV NO_JEVENTS=1

# Default command when the container starts
CMD ["/bin/bash"]
