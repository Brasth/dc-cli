# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers around @devcontainers/cli"
  homepage "https://dc.brasth.com"
  version "0.11.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.11.0/dc-cli-0.11.0-darwin-arm64.tar.gz"
      sha256 "db1ac8436a986bf3711cc8ad478ebce07abb380c2668b24bf7e847e204ec41b5"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.11.0/dc-cli-0.11.0-darwin-amd64.tar.gz"
      sha256 "9c5bd0ca7b345268bb3ccff8e848a150f31740db6e5d94eec0daf5de4c46f6a3"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.11.0/dc-cli-0.11.0-linux-arm64.tar.gz"
      sha256 "09e1491092dd1c52d0684f37174b8e04691dd81c4084a3acbb0f9ca663270d40"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.11.0/dc-cli-0.11.0-linux-amd64.tar.gz"
      sha256 "33e989fe875363d79c787628c4e6c4061735fc7cd9ae8fce463c1fba4eaaf09c"
    end
  end

  def install
    bin.install Dir["bin/*"]
    lib.install Dir["lib/*.sh"]
    pkgshare.install "config/override.json" if File.exist?("config/override.json")
  end

  test do
    assert_match "dc-tui", shell_output("#{bin}/dc-tui --help")
    assert_match "dc-up", shell_output("#{bin}/dc-up --help")
    assert_match "dc-doctor", shell_output("#{bin}/dc-doctor --help")
    assert_match "dc-stats", shell_output("#{bin}/dc-stats --help")
  end

  def caveats
    <<~EOS
      Needs Docker (Colima or Desktop). Official CLI is separate.
      Preferred: standalone via advertised curl --with-cli
        curl -fsSL https://raw.githubusercontent.com/Brasth/dc-cli/main/install.sh | bash -s -- --with-cli
      Explicit npm (exact pin only, never implied by --with-cli):
        bash install.sh --with-cli-npm
      npm pin is empty until docs/qualification/devcontainer-cli-floor.md is signed.

      Port override example (dc-up --ports):
        #{pkgshare}/override.json
      Copy to ~/.config/devcontainer/override.json if you want it.

      One human, one Docker context. Fleet and prune see the whole engine.
    EOS
  end
end
