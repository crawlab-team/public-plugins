FROM golang:1.16 AS build

# add files
COPY . /app

WORKDIR /app

# install plugins
RUN /bin/bash /app/bin/docker-install-plugins.sh

FROM alpine:3.14

COPY --from=build /app/plugins /app/plugins

