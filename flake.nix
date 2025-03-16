{
  description = "inertia - Go adapter for Inertia.js";

  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    flake-utils.inputs = {
      nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      nixpkgs,
      flake-utils,
      ...
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
        ciTestCommand = pkgs.writeScriptBin "ci-test" ''
          go test -count=1 -race -v ./...
        '';
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs =
            with pkgs;
            [
              golangci-lint
              gofumpt
              go
              mockgen
              gotools
              nodejs
            ]
            ++ [ ciTestCommand ];
        };

        env = {
          GOFUMPT_SPLIT_LONG_LINES = "on";
          GOTOOLCHAIN = "local";
        };

        formatter = pkgs.nixfmt-rfc-style;
      }
    );
}
