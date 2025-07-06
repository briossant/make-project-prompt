# flake.nix
{
  description = "Generate LLM prompts with project context (make-project-prompt / mpp)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        lib = pkgs.lib;

        pname = "make-project-prompt";
        version = "0.2.1";

        runtimeDeps = with pkgs; [
          gitMinimal
          tree
          file
        ];

        makeProjectPromptPackage = pkgs.buildGoModule {
          inherit pname version;
          src = ./.;

          # Use the computed hash for the vendor directory
          vendorHash = "sha256-MtzrTRbcG9lMJqN6rcOnwW6xYJbJ3b0J3fuaq9hlEuo=";

          nativeBuildInputs = [ pkgs.makeWrapper ];

          postInstall = ''
            ln -s $out/bin/${pname} $out/bin/mpp
            wrapProgram $out/bin/${pname} --prefix PATH : ${lib.makeBinPath runtimeDeps}
          '';

          meta = with lib; {
            description = "Generates LLM prompts with project context (tree, files)";
            longDescription = ''
              Generates a contextual prompt for LLMs based on the current Git project.
              Includes 'tree' output and content of relevant files (respecting .gitignore,
              with include/exclude options), then copies the result to the clipboard.
              Written in Go for cross-platform compatibility.
              Available as 'make-project-prompt' or 'mpp'.
            '';
            homepage = "https://github.com/briossant/make-project-prompt";
            license = licenses.mit;
            maintainers = with maintainers; [ "brieuc crosson" ];
            platforms = platforms.all;
            mainProgram = "make-project-prompt";
          };
        };

      in
      {
        packages.${pname} = makeProjectPromptPackage;
        packages.default = makeProjectPromptPackage;

        apps.${pname} = {
          type = "app";
          program = "${makeProjectPromptPackage}/bin/${pname}";
        };
        apps.mpp = {
          type = "app";
          program = "${makeProjectPromptPackage}/bin/mpp";
        };
        apps.test = {
          type = "app";
          program = toString (pkgs.writeShellScript "run-tests" ''
            set -e
            cd ${self}
            ${pkgs.go}/bin/go test ./... -v
          '');
        };

        # Linter app
        apps.lint = {
          type = "app";
          program = toString (pkgs.writeShellScript "run-linter" ''
            set -e
            echo "--- Running golangci-lint from flake with auto-fix enabled ---"
            # Execute the linter with auto-fix enabled on the current directory
            # This will work on the actual project files, not the read-only Nix store
            ${pkgs.golangci-lint}/bin/golangci-lint run --fix
          '');
        };

        apps.default = self.apps.${system}.${pname};

        devShells.default = pkgs.mkShell {
          packages = runtimeDeps ++ [
            pkgs.go
            pkgs.gopls
            pkgs.gotools
            pkgs.go-tools
            pkgs.golangci-lint
            makeProjectPromptPackage
          ];
        };
      });
}
