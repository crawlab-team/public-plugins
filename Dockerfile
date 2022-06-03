FROM golang:1.16

# add files
COPY . /app

WORKDIR /app

# install plugins
RUN /bin/bash /app/bin/docker-install-plugins.sh
