{ pkgs, hugo, docsy, dartSass }:

pkgs.mkShell {
  packages = [ hugo pkgs.nodejs_22 dartSass pkgs.helm-docs ];
  DOCSY_THEME = "${docsy}/theme";
  shellHook = ''
    website_root="$(git rev-parse --show-toplevel)/website"
    mkdir -p "$website_root/themes"
    ln -sfn "$DOCSY_THEME" "$website_root/themes/docsy"
    (cd "$website_root" && npm ci)
  '';
}
