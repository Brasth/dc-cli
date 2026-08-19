# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers for Dev Containers and this-folder compose"
  homepage "https://dc.brasth.com"
  version "0.15.1"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.15.1/dc-cli-0.15.1-darwin-arm64.tar.gz"
      sha256 "8cb6569061fc34c61b57bf8f4f8aef2f65b8e7ec2f16d12ddbc8a9559b822144"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.15.1/dc-cli-0.15.1-darwin-amd64.tar.gz"
      sha256 "9c67e259555a13f0df6ef79d55305a34517bd68c0dc7e462a3a28b7696e108ed"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.15.1/dc-cli-0.15.1-linux-arm64.tar.gz"
      sha256 "83c55f6c3a771087875d02ed2a64e3e4aff23e35b80197e4b87d15cf2546405b"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.15.1/dc-cli-0.15.1-linux-amd64.tar.gz"
      sha256 "7eaf7ecbabfb64e3fa2a5de4dda2c5f9812077efe74a32c108a622f8dca422e9"
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
