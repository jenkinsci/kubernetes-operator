{
  description = "Jenkins Kubernetes Operator";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    hugo_099.url = "github:nixos/nixpkgs/d6df226c53d46821bd4773bd7ec3375f30238edb";
    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "flake-utils";
    };
  };

  outputs = { self, nixpkgs, flake-utils, gomod2nix, hugo_099, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        hugo_099_pkgs = hugo_099.legacyPackages.${system};
        operatorVersion = builtins.readFile ./VERSION.txt;
        sdkVersion = ((builtins.fromTOML (builtins.readFile ./config.base.env)).OPERATOR_SDK_VERSION);
        jenkinsLtsVersion = ((builtins.fromTOML (builtins.readFile ./config.base.env)).LATEST_LTS_VERSION);
      in
      {
        # Nix fmt
        formatter = nixpkgs.legacyPackages.${system}.nixpkgs-fmt;

        # shell in nix develop
        devShells.default = nixpkgs.legacyPackages.${system}.mkShell {
          packages = [
            pkgs.gnumake
            pkgs.wget
            pkgs.helm-docs
            pkgs.pre-commit
            pkgs.kind
            pkgs.golangci-lint
            pkgs.go_1_20

            (pkgs.bats.withLibraries (p: [
            p.bats-support
            p.bats-assert
            p.bats-file
            p.bats-detik
            ]))

            (pkgs.writeShellApplication {
              name = "make_matrix";
              runtimeInputs = with pkgs; [ bash gnugrep gawk ];
              text = builtins.readFile ./test/make_matrix_ginkgo.sh;
            })
          ];
          shellHook = ''
            echo Operator Version ${operatorVersion}
            echo Latest Jenkins LTS version: ${jenkinsLtsVersion}
            echo Operator SDK version: ${sdkVersion}
          '';
        };

        # nix shell .#gomod
        devShells.gomod = pkgs.callPackage ./nix/gomod-shell.nix {
          inherit (gomod2nix.legacyPackages.${system}) mkGoEnv gomod2nix;
        };

        # nix shell .#website
        devShells.website = pkgs.callPackage ./nix/website-shell.nix {
          inherit pkgs system hugo_099_pkgs;
        };

        # nix build with gomod2nix
        packages.default = pkgs.callPackage ./nix {
          inherit (gomod2nix.legacyPackages.${system}) buildGoApplication;
        };

        packages.website = import ./nix/website-build.nix {
          inherit pkgs system hugo_099_pkgs;
        };

      }
    );
}
