# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers for Dev Containers and this-folder compose"
  homepage "https://dc.brasth.com"
  version "0.21.0"
  license "MIT"
  depends_on "bash"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.21.0/dc-cli-0.21.0-darwin-arm64.tar.gz"
      sha256 "fd575e29ff0a8bf37e291b288dd64824bb5829f172743748785af5c3595a7e9b"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.21.0/dc-cli-0.21.0-darwin-amd64.tar.gz"
      sha256 "0fa79f36f9185f83a098fedc0155efe87706a5a4ca4e666da36909555c7905a8"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.21.0/dc-cli-0.21.0-linux-arm64.tar.gz"
      sha256 "9b081112bf8de3d280ac5eab8f3c86b2d6ff621df1b8367db43a0875d28d6be3"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.21.0/dc-cli-0.21.0-linux-amd64.tar.gz"
      sha256 "a4d820b4c2d2fc19556ebdbd9f62223eaa5b786897663ad9e48b33a3c4b8d256"
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
