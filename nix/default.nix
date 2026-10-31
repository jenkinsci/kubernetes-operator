{ lib, buildGoModule }:

buildGoModule {
  pname = (builtins.fromTOML (builtins.readFile ../config.base.env)).NAME;
  version = lib.strings.trim (builtins.readFile ../VERSION.txt);
  src = lib.cleanSource ../.;
  vendorHash = "sha256-Q9/z/hPLTEp9s7X7zDbfwDig8bkLO7yUTTEi+aBYGNs=";
  subPackages = [ "cmd" ];

  checkPhase = ''
    runHook preCheck
    RUNNING_TESTS=1 go test -p "$NIX_BUILD_CORES" ./api/... ./internal/... ./pkg/... ./version/...
    runHook postCheck
  '';

  meta.mainProgram = "cmd";
}
