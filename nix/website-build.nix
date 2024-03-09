{ pkgs, system, hugo_099_pkgs }:

let
  buildPackages = [
    hugo_099_pkgs.hugo #hugo pre-v100
    pkgs.nodejs_21 #Node 1.21
    pkgs.nodePackages.autoprefixer
    pkgs.nodePackages.postcss
    pkgs.nodePackages.postcss-cli
  ];
  baseUrl = ((builtins.fromTOML (builtins.readFile ../website/config.toml)).baseURL);
in
pkgs.buildNpmPackage {
  name = "jenkins-kubernetes-operator-website";
  src = ../website;
  version = "0.0.1";
  npmDepsHash = "sha256-NcspVYF+9dCrGxH/cGNhD+TxLZm6ZDX523mKm9smAAA=";
  nativeBuildInputs = buildPackages;
  buildPhase = "npm run build";
  installPhase = "cp -r public $out";
}
