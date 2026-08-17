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
      sha256 "b9a652fc39ea3cfaaebbfa5c12e06d1bed40084b300acea5a3926d13e018d590"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.14.0/dc-cli-0.14.0-darwin-amd64.tar.gz"
      sha256 "9c1a88aed96c5d9bd50bc5439b4a06b29df45b1588f7936904c0ec2272eaae7b"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.14.0/dc-cli-0.14.0-linux-arm64.tar.gz"
      sha256 "a04589fd10d5ed0f199aba8e4e1ddcec7c27de99f14bdebc7b06c4de34593e41"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.14.0/dc-cli-0.14.0-linux-amd64.tar.gz"
      sha256 "7223ef4af297e022dd3782fa8b668ad5b6ab17ee28c55cc9bb8f3284042742f4"
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
