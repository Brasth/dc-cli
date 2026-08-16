# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers around @devcontainers/cli"
  homepage "https://dc.brasth.com"
  version "0.10.3"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.3/dc-cli-0.10.3-darwin-arm64.tar.gz"
      sha256 "a957bd5ac5455c932320792562b56c383d2c204d8f17707c83689b91d6dbe1ef"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.3/dc-cli-0.10.3-darwin-amd64.tar.gz"
      sha256 "be798576eff6fcc9bbc775ada2cfea28906edaefdb8090084d1e74b22bc6fa9f"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.3/dc-cli-0.10.3-linux-arm64.tar.gz"
      sha256 "7989270019802aab40d0cd40f4378adc178a4b73058468b58aa49580661495ce"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.3/dc-cli-0.10.3-linux-amd64.tar.gz"
      sha256 "65a456d881805beb92c6e0ed8a5919fdd8e60ab78f8d082a09caaa95fe412013"
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
