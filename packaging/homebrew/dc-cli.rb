# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers for Dev Containers and this-folder compose"
  homepage "https://dc.brasth.com"
  version "0.14.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.14.0/dc-cli-0.14.0-darwin-arm64.tar.gz"
      sha256 "1768e7139a29ec91009fa65d6ef84a30419f0c3176a3a8638f2165f79e953a49"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.14.0/dc-cli-0.14.0-darwin-amd64.tar.gz"
      sha256 "56189b2e92df1dd0dae5338a84419ae4374856af50425de128fd530186e404de"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.14.0/dc-cli-0.14.0-linux-arm64.tar.gz"
      sha256 "1d22b829a4579e2a929bd5cebe680e23a510f537104ab23d68c3cd958da73838"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.14.0/dc-cli-0.14.0-linux-amd64.tar.gz"
      sha256 "b2b0cca707651f1d3efb95b614894a5f0ddb5794c7a90b34ee013e577f41c837"
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
