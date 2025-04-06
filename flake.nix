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
        version = "0.1.1";

        scriptFile = ./gen_prompt.sh;

        scriptDeps = with pkgs; [
          gitMinimal
          tree
          xsel
          file
          coreutils
          bash
        ];

        makeProjectPromptPackage = pkgs.stdenv.mkDerivation {
          inherit pname version;
          src = ./.;

          nativeBuildInputs = [ pkgs.makeWrapper ];
          dontBuild = true;

          installPhase = ''
            runHook preInstall
            install -Dm 755 ${scriptFile} $out/bin/${pname}
            ln -s $out/bin/${pname} $out/bin/mpp
            wrapProgram $out/bin/${pname} --prefix PATH : ${lib.makeBinPath scriptDeps}
            runHook postInstall
          '';

          meta = with lib; {
            description = "Generates LLM prompts with project context (tree, files)";
            longDescription = ''
              Generates a contextual prompt for LLMs based on the current Git project.
              Includes 'tree' output and content of relevant files (respecting .gitignore,
              with include/exclude options), then copies the result to the clipboard via xsel.
              Available as 'make-project-prompt' or 'mpp'.
            '';
            homepage = "https://github.com/briossant/make-project-prompt";
            license = licenses.mit;
            maintainers = with maintainers; [ "briossant" ];
            platforms = platforms.unix;
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

        apps.default = self.apps.${system}.${pname};

        devShells.default = pkgs.mkShell {
          packages = scriptDeps ++ [
            pkgs.shellcheck
            pkgs.bashInteractive
            makeProjectPromptPackage
          ];
        };
      });
}
