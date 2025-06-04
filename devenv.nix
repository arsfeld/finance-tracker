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
    air
    supabase-cli
  ];

  languages.go.enable = true;
}
