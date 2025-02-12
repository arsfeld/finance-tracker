{
  pkgs,
  lib,
  config,
  inputs,
  ...
}: {
  # https://devenv.sh/basics/
  env.GREET = "devenv";

  env.API_URL = "http://localhost:5150";

  # https://devenv.sh/packages/
  packages = with pkgs; [
    git
    gcc
    cargo-watch
    cargo-edit
    pnpm
    nodePackages.prettier
    nodejs
    openssl.dev
    pkg-config
  ];

  languages.rust.enable = true;

  # https://devenv.sh/languages/
  # languages.rust.enable = true;

  # https://devenv.sh/processes/
  processes.cargo-watch.exec = "cargo-watch -- cargo loco start";

  processes.pnpm-dev.exec = "cd frontend && pnpm dev";

  # https://devenv.sh/services/
  # services.postgres.enable = true;

  # https://devenv.sh/tasks/
  scripts = {
    "start-frontend".exec = "cd frontend && pnpm dev";
    "start-backend".exec = "cargo loco start --server-and-worker";

    "build-frontend".exec = "cd frontend && pnpm build";
    "build-backend".exec = "cargo loco build";
  };

  # https://devenv.sh/tests/
  enterTest = ''
    echo "Running tests"
    git --version | grep --color=auto "${pkgs.git.version}"
  '';

  # https://devenv.sh/pre-commit-hooks/
  # pre-commit.hooks.shellcheck.enable = true;

  # See full reference at https://devenv.sh/reference/options/
}
