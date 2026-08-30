{
  description = "Telegram bot for running a daily roulette game in group chats";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs =
    { self, nixpkgs }:
    let
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs systems (system: f system nixpkgs.legacyPackages.${system});

      version =
        let
          d = self.lastModifiedDate or "19700101000000";
          date = "${builtins.substring 0 4 d}-${builtins.substring 4 2 d}-${builtins.substring 6 2 d}";
          rev = self.shortRev or self.dirtyShortRev or "unknown";
        in
        "0-unstable-${date}-${rev}";
    in
    {
      overlays.default = final: _prev: {
        telegram-chat-bot = final.callPackage ./package.nix { inherit version; };
      };

      packages = forAllSystems (
        _system: pkgs: rec {
          telegram-chat-bot = pkgs.callPackage ./package.nix { inherit version; };
          default = telegram-chat-bot;
        }
      );

      devShells = forAllSystems (
        _system: pkgs: {
          default = pkgs.mkShell {
            packages = [
              pkgs.go_1_27
              pkgs.gopls
              pkgs.gotools
              pkgs.sqlc
              pkgs.sqlite
              pkgs.python3
            ];
          };
        }
      );

      checks = forAllSystems (
        system: _pkgs: {
          build = self.packages.${system}.telegram-chat-bot;
        }
      );

      formatter = forAllSystems (_system: pkgs: pkgs.nixfmt-tree);
    };
}
