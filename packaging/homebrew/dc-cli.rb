# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers around @devcontainers/cli"
  homepage "https://dc.brasth.com"
  version "0.8.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.8.0/dc-cli-0.8.0-darwin-arm64.tar.gz"
      sha256 "76377ceadc1c4099c04c76f07ecea03f6e6e1b18887a8085310fc97c2e7a5331"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.8.0/dc-cli-0.8.0-darwin-amd64.tar.gz"
      sha256 "a00c44f18f481ca2fb74d8dd038ddd24ef617573702d54007da7c0c6e282ce66"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.8.0/dc-cli-0.8.0-linux-arm64.tar.gz"
      sha256 "a40597798fa5d994bb061ee8aceadeae93c8eb4575da9ea0829b3fd6a3efc861"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.8.0/dc-cli-0.8.0-linux-amd64.tar.gz"
      sha256 "991ca2808af8b77bfae4a02695d7ae21334b8e1cc59ffa2242b8a41d4b9ff013"
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
