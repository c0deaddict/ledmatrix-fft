{ pkgs, lib, config, ... }:

with lib;

let

  cfg = config.services.ledmatrix-fft;

  args = [ "server" ] ++ cfg.extraArgs;

in
{

  options.services.ledmatrix-fft = {
    enable = mkEnableOption "ledmatrix-fft";

    package = mkOption {
      type = types.package;
      default = pkgs.callPackage ../pkgs/ledmatrix-fft.nix { };
    };

    extraArgs = mkOption {
      type = with types; listOf str;
      default = [ ];
      description = "Extra arguments to pass to ledmatrix-fft.";
    };

    cli.enable = mkEnableOption "install cli in home.packages";
  };

  config = mkIf cfg.enable {
    systemd.user.services.ledmatrix-fft = {
      Unit = {
        Description = "Ledmatrix FFT";
      };

      Service = {
        Type = "simple";
        ExecStart = "${cfg.package}/bin/ledmatrix-fft ${concatStringsSep " " args}";
        Restart = "on-failure";
        RestartSec = 3;
      };

      Install = { WantedBy = [ "graphical-session.target" ]; };
    };

    home.packages = if cfg.cli.enable then [ cfg.package ] else [ ];
  };
}
