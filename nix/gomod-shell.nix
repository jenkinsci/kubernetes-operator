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
, go ? pkgs.go_1_26
, golangci-lint ? pkgs.golangci-lint
}:

let
  goEnv = mkGoEnv {
    pwd = ../.;
    modules = ./gomod2nix.toml;
    inherit go;
  };
in
pkgs.mkShell {
  packages = [
    go
    golangci-lint
    goEnv
    gomod2nix
  ];
}
