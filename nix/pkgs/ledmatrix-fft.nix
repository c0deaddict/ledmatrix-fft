{ lib, buildGoModule, makeWrapper, cava }:

buildGoModule rec {
  pname = "ledmatrix-fft";
  version = "0.0.1";

  src = ../..;

  vendorSha256 = "sha256-Xw8jHPVmNicbc/f+iex7QhR/mLkOzxPexVLqy+/GVOI=";

  subPackages = [ "cmd/ledmatrix-fft" ];

  buildInputs = [ ];

  nativeBuildInputs = [ makeWrapper ];

  postInstall = ''
    mkdir $out/assets
    cp cava.config $out/assets/

    wrapProgram $out/bin/ledmatrix-fft \
      --prefix PATH : ${lib.makeBinPath [cava]} \
      --chdir $out/assets
  '';

  meta = with lib; {
    description = "Ledmatrix FFT with Spotify track information";
    homepage = "https://github.com/c0deaddict/ledmatrix-fft";
    license = licenses.mit;
    maintainers = with maintainers; [ c0deaddict ];
  };
}
