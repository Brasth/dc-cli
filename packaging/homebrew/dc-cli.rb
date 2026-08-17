# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers for Dev Containers and this-folder compose"
  homepage "https://dc.brasth.com"
  version "0.14.1"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.14.1/dc-cli-0.14.1-darwin-arm64.tar.gz"
      sha256 "4320e804effb64a183af7fe7af18d3b922abbcd282f7b25f9fd59e28f8ba7499"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.14.1/dc-cli-0.14.1-darwin-amd64.tar.gz"
      sha256 "c2b2bfa3eb5c0c5385ca66c37d2f4e7c4335b737dd62683c815dc632e4013959"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.14.1/dc-cli-0.14.1-linux-arm64.tar.gz"
      sha256 "09f914bc24e0794c404325647ef9200ac60fa186d4d60d1c61df0db299374ba2"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.14.1/dc-cli-0.14.1-linux-amd64.tar.gz"
      sha256 "0a136255eb80ec24819047f7f28c1f8243c5a233a923da74d7a2c0bd67977616"
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
