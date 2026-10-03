# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers for Dev Containers and this-folder compose"
  homepage "https://dc.brasth.com"
  version "0.22.0"
  license "MIT"
  depends_on "bash"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.22.0/dc-cli-0.22.0-darwin-arm64.tar.gz"
      sha256 "55fe0ba35efa74c79de76567d8bb8bc1397a623b61ac57395ba36a8043ca275c"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.22.0/dc-cli-0.22.0-darwin-amd64.tar.gz"
      sha256 "44f56c25f21333a1d6125b56b7650ca419af6b647b6d0f2ae52e59cbf5b3691c"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.22.0/dc-cli-0.22.0-linux-arm64.tar.gz"
      sha256 "35da5eb73517e867eb8d0d24f732040d5400c1b963c6b5a629e0830f6b81e089"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.22.0/dc-cli-0.22.0-linux-amd64.tar.gz"
      sha256 "45769bcea6947ecfb703ad203f5c76767a91d2c157aae3aed90a03237301495a"
    end
  end

  def install
    bin.install Dir["bin/*"]
    lib.install Dir["lib/*.sh"]
    pkgshare.install "config/override.json" if File.exist?("config/override.json")
  end

  test do
    assert_match "dc up", shell_output("#{bin}/dc --help")
    assert_match "dc-tui", shell_output("#{bin}/dc-tui --help")
    assert_match "dc-up", shell_output("#{bin}/dc-up --help")
    assert_match "dc-doctor", shell_output("#{bin}/dc-doctor --help")
    assert_match "dc-stats", shell_output("#{bin}/dc-stats --help")
    assert_match "dc-net", shell_output("#{bin}/dc-net --help")
    assert_match "dc-engine", shell_output("#{bin}/dc-engine --help")
    assert_match "dc-try", shell_output("#{bin}/dc-try --help")
    assert_match "dc-inspect", shell_output("#{bin}/dc-inspect --help")
    assert_match "dc-actions", shell_output("#{bin}/dc-actions --help")
    assert_match "0.22.0", shell_output("#{bin}/dc-actions --version")
  end

  def caveats
    <<~EOS
      Helpers need Bash 4+ (Homebrew bash on macOS).
      Needs Docker (Colima or Desktop — one live engine). Official CLI is required only for
      Dev Container folders. Compose-only folders use docker compose or docker-compose via dc-up.
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
