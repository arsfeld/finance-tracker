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
    go
    just
  ];

  languages.go.enable = true;
}
