# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers around @devcontainers/cli"
  homepage "https://dc.brasth.com"
  version "0.12.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.12.0/dc-cli-0.12.0-darwin-arm64.tar.gz"
      sha256 "63cff330ad68a6fc379a85c48fdcfca7aefd99dddf5b6fbddf180759a0806edb"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.12.0/dc-cli-0.12.0-darwin-amd64.tar.gz"
      sha256 "d97c4916ab289eec303680abe2411ab7d1dd0d8b2a10dd3a45977e951e778ce6"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.12.0/dc-cli-0.12.0-linux-arm64.tar.gz"
      sha256 "79eb63977fbc2eccb3093fa19965bcce481f1238d9ea0291da9eac3afe76a3f9"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.12.0/dc-cli-0.12.0-linux-amd64.tar.gz"
      sha256 "36cacc7ce18defc0e5a3767ccd8907698fff1fd9f8a3d66a48b702dfeb0b3545"
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
