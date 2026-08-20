# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers for Dev Containers and this-folder compose"
  homepage "https://dc.brasth.com"
  version "0.16.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.16.0/dc-cli-0.16.0-darwin-arm64.tar.gz"
      sha256 "e5e29445a134eac3fe637ab72bf4d42bd3edecb96b286add5dc5816754d3a5f9"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.16.0/dc-cli-0.16.0-darwin-amd64.tar.gz"
      sha256 "ad763c75c56c63ec37324cfc6967879682476dd00a2ae477281a36e7fab4c56d"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.16.0/dc-cli-0.16.0-linux-arm64.tar.gz"
      sha256 "f054a776268ce92436f24cca0125060123ee8e44816f73c37ddbd32e4151b82c"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.16.0/dc-cli-0.16.0-linux-amd64.tar.gz"
      sha256 "6e6967562af52e1e5ccae11435759fc3a4a16c5ad99155a8f171b3df5210c702"
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
    assert_match "dc-net", shell_output("#{bin}/dc-net --help")
    assert_match "dc-engine", shell_output("#{bin}/dc-engine --help")
    assert_match "dc-try", shell_output("#{bin}/dc-try --help")
  end

  def caveats
    <<~EOS
      Needs Docker (Colima or Desktop — one live engine). Official CLI is required only for
      Dev Container folders. Compose-only folders use docker compose via dc-up.
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
