{ pkgs, system, hugo_099_pkgs }:

let
  devShellPackages = [
    hugo_099_pkgs.hugo #hugo pre-v100
    pkgs.nodejs_22 #Node 1.22
    pkgs.helm-docs
  ];
  baseUrl = ((builtins.fromTOML (builtins.readFile ../website/config.toml)).baseURL);
in
pkgs.mkShell {
  packages = devShellPackages;
  shellHook = ''
    npm install --save-dev
    npm list
  '';
  BASE_URL = "${baseUrl}";
}
