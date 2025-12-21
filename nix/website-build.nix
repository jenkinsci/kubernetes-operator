{ pkgs, system, hugo_099_pkgs }:

let
  buildPackages = [
    hugo_099_pkgs.hugo #hugo pre-v100
    pkgs.nodejs_22 #Node 1.22
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
  npmDepsHash = "sha256-x/OPri4wO0KFPJ2ZGmLTl/BdHUH4qlj0c5UsqiNC8EM=";
  nativeBuildInputs = buildPackages;
  buildPhase = "${pkgs.nodejs_22}/bin/npm run build";
  installPhase = "cp -r public $out";
  BASE_URL = "${baseUrl}";
}
