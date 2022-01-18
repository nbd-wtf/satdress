{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      rec {
        packages = flake-utils.lib.flattenTree {
          satdress = pkgs.buildGoModule {
            pname = "satdress";
            version = "v0.5.0";
            src = self;

            vendorSha256 = "0xz93pmpvqg3wa9ixs7hkaqgmybnqgllf2991nrabkall1dckamh";
          };
        };

        defaultPackage = packages.satdress;

        devShell = pkgs.mkShell {
          buildInputs = with pkgs; [ go packages.satdress ];

          shellHook = ''
            echo "Dev shell ready"
          '';
        };

      });
}
