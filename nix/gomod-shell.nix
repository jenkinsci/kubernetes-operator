{ pkgs ? (
    let
      inherit (builtins) fetchTree fromJSON readFile;
      inherit ((fromJSON (readFile ./flake.lock)).nodes) nixpkgs gomod2nix;
    in
    import (fetchTree nixpkgs.locked) {
      overlays = [
        (import "${fetchTree gomod2nix.locked}/overlay.nix")
      ];
    }
  )
, mkGoEnv ? pkgs.mkGoEnv
, gomod2nix ? pkgs.gomod2nix
, go22 ? pkgs.go_1_22
, golangci-lint ? pkgs.golangci-lint
}:

let
  goEnv = mkGoEnv { pwd = ../.; };
in
pkgs.mkShell {
  packages = [
    go22
    golangci-lint
    goEnv
    gomod2nix
  ];
}
