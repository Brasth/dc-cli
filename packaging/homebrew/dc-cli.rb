# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers for Dev Containers and this-folder compose"
  homepage "https://dc.brasth.com"
  version "0.20.0"
  license "MIT"
  depends_on "bash"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.20.0/dc-cli-0.20.0-darwin-arm64.tar.gz"
      sha256 "3f1b39221a33172092c0acdbd9f54db37fbf2b9fd2a157a3b93fa786e2b4bd41"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.20.0/dc-cli-0.20.0-darwin-amd64.tar.gz"
      sha256 "6682e905c7d4aef1267b63f37737e2bf8b301959b1806387fd591e657b0b2b30"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.20.0/dc-cli-0.20.0-linux-arm64.tar.gz"
      sha256 "044325125efdba0bccc152680a23a29abbc3128dedd2fcfb33cad92c50fec70b"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.20.0/dc-cli-0.20.0-linux-amd64.tar.gz"
      sha256 "733856234b473e9c048dfa90f9a6285d6be3bc314d57952ed3eddb2789bb3458"
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
