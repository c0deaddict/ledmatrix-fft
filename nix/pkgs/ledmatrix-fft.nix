{ lib, buildGoModule }:

buildGoModule rec {
  pname = "ledmatrix-fft";
  version = "0.0.1";

  src = ../..;

  vendorSha256 = "sha256-a6pK0uJRl+ShOPrjbe2A5CsJ5IKEtpzDKQK33v/Fzho=";

  buildInputs = [ ];

  meta = with lib; {
    description = "Ledmatrix FFT with Spotify track information";
    homepage = "https://github.com/c0deaddict/ledmatrix-fft";
    license = licenses.mit;
    maintainers = with maintainers; [ c0deaddict ];
  };
}
