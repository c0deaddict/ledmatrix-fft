{ pkgs, lib, config, ... }:

with lib;

let

  cfg = config.services.ledmatrix-fft;
  format = pkgs.formats.json { };
  configFile = format.generate "config.json" cfg.settings;

in
{

  options.services.ledmatrix-fft = {
    enable = mkEnableOption "ledmatrix-fft";

    package = mkOption {
      type = types.package;
      default = pkgs.callPackage ../pkgs/ledmatrix-fft.nix { };
    };

    settings = mkOption {
      default = { };
      type = format.type;
    };
  };

  config = mkIf cfg.enable {
    systemd.user.services.ledmatrix-fft = {
      Unit = {
        Description = "Ledmatrix FFT";
      };

      Service = {
        Type = "simple";
        ExecStart = "${cfg.package}/bin/ledmatrix-fft -config ${configFile}";
        Restart = "on-failure";
        RestartSec = 3;
      };

      Install = { WantedBy = [ "graphical-session.target" ]; };
    };
  };
}
