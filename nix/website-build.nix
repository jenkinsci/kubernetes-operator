{ pkgs, system, hugo_099_pkgs }:

let
  buildPackages = [
    hugo_099_pkgs.hugo #hugo pre-v100
    pkgs.nodejs_22 #Node 1.22
  ];
  baseUrl = ((builtins.fromTOML (builtins.readFile ../website/config.toml)).baseURL);
in
pkgs.buildNpmPackage {
  name = "jenkins-kubernetes-operator-website";
  src = ../website;
  version = "0.0.2";
  npmDepsHash = "sha256-OaLvJNwViN7q2E7eHVn64eYTQZ7kVDOULx1fdBcliF8=";
  nativeBuildInputs = buildPackages;
  buildPhase = "${pkgs.nodejs_22}/bin/npm run build";
  installPhase = "cp -r public $out";
  BASE_URL = "${baseUrl}";
}
