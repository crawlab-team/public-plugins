#!/bin/bash

function install_plugin() {
  # plugins executables directory
  local bin_path="/app/plugins/bin"
  if [ -d $bin_path ]; then
    :
  else
    mkdir -p "$bin_path"
  fi

  # variables
  local name=$1
  local plugin_path="/app/plugins/${name}"

  # echo variables
  echo "name=$name"
  echo "plugin_path=$plugin_path"

  # build
  cd "$plugin_path" && go build -o "${bin_path}/${name}"

  # ensure executable
  chmod +x "${bin_path}/${name}"
}

mkdir -p /app/plugins

for name in `ls | grep plugin-`; do
  cp -r $name /app/plugins/
done

ls -l /app/plugins

for name in `ls | grep plugin-`; do
  install_plugin "$name"
done
