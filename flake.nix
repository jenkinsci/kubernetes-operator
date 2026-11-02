{
  description = "Jenkins Kubernetes Operator";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs?ref=nixos-26.05";
    nixpkgs-rolling.url = "github:nixos/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    hugo_099.url = "github:nixos/nixpkgs/d6df226c53d46821bd4773bd7ec3375f30238edb";
  };

  outputs =
    inputs:
    inputs.flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = inputs.nixpkgs.legacyPackages.${system};
        rolling = inputs.nixpkgs-rolling.legacyPackages.${system};
        hugo_099_pkgs = inputs.hugo_099.legacyPackages.${system};
        toolVersions = builtins.fromTOML (builtins.readFile ./config.base.env);
        go = pkgs.${"go_" + builtins.replaceStrings [ "." ] [ "_" ] toolVersions.GO_VERSION};
        operatorVersion = builtins.readFile ./VERSION.txt;
        sdkVersion = toolVersions.OPERATOR_SDK_VERSION;
        jenkinsLtsVersion = toolVersions.LATEST_LTS_VERSION;
        updateNixHashes = pkgs.writeShellApplication {
          name = "update-nix-hashes";
          runtimeInputs = [ pkgs.nix-update ];
          text = ''
            nix-update --flake --version=skip --override-filename nix/default.nix default
            nix-update --flake --version=skip --override-filename nix/website-build.nix website
          '';
        };
      in
      {
        # Nix fmt
        formatter = pkgs.nixpkgs-fmt;

        apps.update-nix-hashes = {
          type = "app";
          program = pkgs.lib.getExe updateNixHashes;
        };

        # shell in nix develop
        devShells.default = pkgs.mkShell {
          packages = [
            pkgs.gnumake
            pkgs.wget
            pkgs.helm-docs
            pkgs.pre-commit
            pkgs.kind
            pkgs.golangci-lint
            go # 1.26
            rolling.operator-sdk # 1.39.2

            (pkgs.bats.withLibraries (p: [
              p.bats-support
              p.bats-assert
              p.bats-file
              p.bats-detik
            ]))

            (pkgs.writeShellApplication {
              name = "make_matrix";
              runtimeInputs = with pkgs; [
                bash
                gnugrep
                gawk
              ];
              text = builtins.readFile ./test/make_matrix_ginkgo.sh;
            })
          ];
          shellHook = ''
            echo Operator Version ${operatorVersion}
            echo Latest Jenkins LTS version: ${jenkinsLtsVersion}
            echo Operator SDK version: ${sdkVersion}
          '';
        };

        # nix shell .#website
        devShells.website = pkgs.callPackage ./nix/website-shell.nix {
          inherit pkgs system hugo_099_pkgs;
        };

        # nix build
        packages.default = pkgs.callPackage ./nix {
          buildGoModule = pkgs.buildGoModule.override { inherit go; };
        };

        packages.website = import ./nix/website-build.nix {
          inherit pkgs system hugo_099_pkgs;
        };

      }
    );
}
