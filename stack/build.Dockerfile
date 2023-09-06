FROM ubuntu:jammy

ARG sources
ARG packages
ARG package_args='--no-install-recommends'

RUN echo "$sources" > /etc/apt/sources.list && \
  echo "Package: $packages\nPin: release c=multiverse\nPin-Priority: -1\n\nPackage: $packages\nPin: release c=restricted\nPin-Priority: -1\n" > /etc/apt/preferences && \
  echo "debconf debconf/frontend select noninteractive" | debconf-set-selections && \
  export DEBIAN_FRONTEND=noninteractive && \
  apt-get -y $package_args update && \
  apt-get -y $package_args upgrade && \
  apt-get -y $package_args install locales && \
  locale-gen en_US.UTF-8 && \
  update-locale LANG=en_US.UTF-8 LANGUAGE=en_US.UTF-8 LC_ALL=en_US.UTF-8 && \
  apt-get -y $package_args install $packages && \
  rm -rf /var/lib/apt/lists/* /tmp/* /etc/apt/preferences && \
  for path in /workspace /workspace/source-ws /workspace/source; do git config --system --add safe.directory "${path}"; done
