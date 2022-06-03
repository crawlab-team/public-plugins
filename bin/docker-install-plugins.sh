#!/bin/bash

function install_plugin() {
  # plugins executables directory
  local bin_path="/app/plugins/bin"
  if [ -d $bin_path ]; then
    :
  else
    mkdir -p "$bin_path"
  fi

  # plugin name
  local name=$1
  local url="https://github.com/crawlab-team/${name}"
  local plugins_root_path="/app/plugins"
  local plugin_path="${plugins_root_path}/${name}"
  cp -rf $name $plugin_path
  cd "$plugin_path" && go build -o "${bin_path}/${name}"
  chmod +x "${bin_path}/${name}"
}

for name in `ls | grep plugin-`; do
  install_plugin "$name"
done
