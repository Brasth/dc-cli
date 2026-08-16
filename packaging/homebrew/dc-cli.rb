# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers around @devcontainers/cli"
  homepage "https://dc.brasth.com"
  version "0.9.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.9.0/dc-cli-0.9.0-darwin-arm64.tar.gz"
      sha256 "ee3509925052e830fc002f7a79459b61367ca39de042d6be14b0d178b27ef7b3"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.9.0/dc-cli-0.9.0-darwin-amd64.tar.gz"
      sha256 "c4defd7571e8c40e8d2ff9facae327e48bd89537842923f9cc160fa5eb8c2823"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.9.0/dc-cli-0.9.0-linux-arm64.tar.gz"
      sha256 "b27c6b70dfe3722e5391a2f17440fcf0af0dffd14b9f1d493e4ae053616f69ad"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.9.0/dc-cli-0.9.0-linux-amd64.tar.gz"
      sha256 "1d9888c7fd845196c07e1cf6b1755a5178875d073445dcb39346f21731e9412b"
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
