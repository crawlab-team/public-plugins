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
  local plugins_root_path="/app/plugins"
  local plugin_path="${plugins_root_path}/${name}"

  # echo variables
  echo "name=$name"
  echo "plugins_root_path=$plugins_root_path"
  echo "plugin_path=$plugin_path"

  # copy dir
  cp -rf $name "${plugins_root_path}/"

  # build
  cd "$plugin_path" && go build -o "${bin_path}/${name}"

  # ensure executable
  chmod +x "${bin_path}/${name}"

  # echo
  ls -l "${plugins_root_path}"
}

for name in `ls | grep plugin-`; do
  install_plugin "$name"
done
