{
  description = "Ledmatrix FFT with Spotify track information";

  inputs = { nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable"; };

  outputs = inputs@{ self, nixpkgs, ... }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f system);
    in
    {
      overlay = final: prev: import ./nix/pkgs/default.nix { pkgs = final; };

      hmModules.ledmatrix-fft = import ./nix/hm-modules/ledmatrix-fft.nix;
      hmModule = self.nixosModules.ledmatrix-fft;

      packages = forAllSystems (system:
        import ./nix/pkgs/default.nix rec {
          pkgs = import nixpkgs { inherit system; };
        });
      defaultPackage = forAllSystems (system: self.packages.${system}.ledmatrix-fft);
    };
}
