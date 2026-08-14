{
  description = "hausv-org dev environment";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "aarch64-darwin" "x86_64-darwin" "aarch64-linux" "x86_64-linux" ];
      forAllSystems = f:
        nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          # Mirrors the CI toolchain in .github/workflows/ci.yml: Go from
          # go.mod and Node 24. Kein fish mehr — scripts/*.sh laufen unter der
          # bash, die jedes System ohnehin mitbringt (HAUSV-427).
          #
          # nixpkgs-unstable currently ships the exact Go in go.mod (1.26.6).
          # If the two ever drift, Go's own GOTOOLCHAIN fetches the pinned
          # version on demand, so builds stay correct either way.
          packages = with pkgs; [
            go
            nodejs_24

            # Linters kept alongside the toolchain so they are versioned with it
            # rather than living only in a hand-managed ~/go/bin.
            gotools # goimports
            go-tools # staticcheck
            govulncheck
            gitleaks
          ];
        };
      });
    };
}
