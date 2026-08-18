# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers for Dev Containers and this-folder compose"
  homepage "https://dc.brasth.com"
  version "0.15.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.15.0/dc-cli-0.15.0-darwin-arm64.tar.gz"
      sha256 "153b62b8441c89adf45d314cb874e182449a1a159e1b2cbdea3d5f07a899d7e1"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.15.0/dc-cli-0.15.0-darwin-amd64.tar.gz"
      sha256 "0db05584528b8e5621a4bccf7cb3340d5f41264ffdc815f3116d5e761caf4722"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.15.0/dc-cli-0.15.0-linux-arm64.tar.gz"
      sha256 "61389da4a5c0b31a6fe090919141997f19a084add4fe30d33b0f21afbd997270"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.15.0/dc-cli-0.15.0-linux-amd64.tar.gz"
      sha256 "e05858f891c66e564c1defde976d069952ef4c2f87231f8bbf32989009c59e5c"
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
