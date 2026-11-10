{ pkgs, hugo, docsy, dartSass }:

let
  buildPackages = [
    hugo
    pkgs.nodejs_22
    dartSass
  ];
  baseUrl = ((builtins.fromTOML (builtins.readFile ../website/config.toml)).baseURL);
in
pkgs.buildNpmPackage {
  name = "jenkins-kubernetes-operator-website";
  src = ../website;
  version = "0.0.2";
  npmDepsHash = "sha256-jo/hC49d4k+XYFl1RjexsErKhpwzjsR5BZMzm96RXLI=";
  nativeBuildInputs = buildPackages;
  postPatch = ''
    mkdir -p themes
    ln -s ${docsy}/theme themes/docsy
  '';
  buildPhase = "${pkgs.nodejs_22}/bin/npm run build";
  installPhase = "cp -r public $out";
  BASE_URL = "${baseUrl}";
}
