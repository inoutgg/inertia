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
          golangci-lint run ./...
          go test -count=1 -race -v ./...
        '';
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs =
            with pkgs;
            [
              go
              gotools
              golangci-lint
              mockgen
              typos
              just

              # Examples
              nodejs
              air
            ]
            ++ [ ciTestCommand ];
        };

        formatter = pkgs.nixfmt-rfc-style;
      }
    );
}
