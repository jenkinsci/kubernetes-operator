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
  version = "0.0.2";
  npmDepsHash = "sha256-VrHuyqTPUzVJSjah+BWfg7R9yiarJQ2MDvEdqkOWddM=";
  nativeBuildInputs = buildPackages;
  buildPhase = "${pkgs.nodejs_21}/bin/npm run build";
  installPhase = "cp -r public $out";
  BASE_URL = "${baseUrl}";
}
