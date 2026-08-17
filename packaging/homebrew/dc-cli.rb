# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers for Dev Containers and this-folder compose"
  homepage "https://dc.brasth.com"
  version "0.13.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.13.0/dc-cli-0.13.0-darwin-arm64.tar.gz"
      sha256 "c40025fe74da2bfff98fe3f7eca0e398b18a401256af2dc047ca7d9060dc0b8f"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.13.0/dc-cli-0.13.0-darwin-amd64.tar.gz"
      sha256 "82c80d028c4440c58b7232b4f681f9b62e51706277a3995eb83ee2b41e5ea0af"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.13.0/dc-cli-0.13.0-linux-arm64.tar.gz"
      sha256 "2a3443cb1373db9e1fd481a84d6729897f80cfdd5467953103c9b043466eb027"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.13.0/dc-cli-0.13.0-linux-amd64.tar.gz"
      sha256 "fcd802ad944e6504b00a0a737221dfee0affb0261f093c18bc73cd99c89e64d2"
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
      Needs Docker (Colima or Desktop). Official CLI is required only for
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
