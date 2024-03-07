{ pkgs, system, hugo_099_pkgs }:

let
  devShellPackages = [
    pkgs.go
    hugo_099_pkgs.hugo #hugo pre-v100
    # docsy theme needs some node dep
    pkgs.nodejs_21 #Node 1.21
    pkgs.nodePackages.postcss
    pkgs.nodePackages.postcss-cli
    pkgs.nodePackages.postcss-cli
  ];
in
pkgs.mkShell {
  packages = devShellPackages;
  shellHook = ''
  '';
}
