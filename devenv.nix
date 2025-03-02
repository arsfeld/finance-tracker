{
  pkgs,
  lib,
  config,
  inputs,
  ...
}: {
  # https://devenv.sh/packages/
  packages = with pkgs; [
    git
    gcc
    cargo-watch
    cargo-edit
    cargo-machete
    cargo-bloat
    openssl.dev
    pkg-config
    rustfmt
    clippy
  ];

  languages.rust.enable = true;

  git-hooks.hooks = {
    rustfmt.enable = true;
    clippy.enable = true;
    clippy.settings.allFeatures = true;
  };
}
