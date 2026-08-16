# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers around @devcontainers/cli"
  homepage "https://dc.brasth.com"
  version "0.10.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.0/dc-cli-0.10.0-darwin-arm64.tar.gz"
      sha256 "5b4936808ecd4089d9edccecdcc4fbb6d1ce570c033a6d46072cde4e014731c3"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.0/dc-cli-0.10.0-darwin-amd64.tar.gz"
      sha256 "3862c0d1c4ca0a4c69bb1d53822bc1ff9d960da729d29c326d76126d758d40b6"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.0/dc-cli-0.10.0-linux-arm64.tar.gz"
      sha256 "5ab1ec0b87f8ff562937f040233161af1b67e9c2ff568ee14b6f877bb7b6f0ce"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.0/dc-cli-0.10.0-linux-amd64.tar.gz"
      sha256 "883a3523f0887369d796c85623dd23eed90865e53b309d48a2c470cec48a4c08"
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
