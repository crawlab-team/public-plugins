FROM golang:1.16

# add files
COPY ./bin /app/bin

# install plugins
RUN /bin/bash /app/bin/docker-install-plugins.sh
