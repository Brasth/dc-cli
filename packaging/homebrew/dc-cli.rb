# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/dc-common.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers around @devcontainers/cli"
  homepage "https://dc.brasth.com"
  version "0.7.1"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.7.1/dc-cli-0.7.1-darwin-arm64.tar.gz"
      sha256 "e31f5a9df636f5ca6593a80c2c7e0af9affc64c5e8b1dbe9bb9d91a29c2cac76"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.7.1/dc-cli-0.7.1-darwin-amd64.tar.gz"
      sha256 "bb924767fc6cc86e04bbc45a63f74e7815d4ed11a1840f1bdbcf4e44b13dcf12"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.7.1/dc-cli-0.7.1-linux-arm64.tar.gz"
      sha256 "703a47ccc2fd389bd0c73fdf6b13a7b03155f560dd6c45ef74f0f187816692c6"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.7.1/dc-cli-0.7.1-linux-amd64.tar.gz"
      sha256 "73107d612f6690d9799672b6415ab78e07d5cb3bac2813669623225b98c4fe70"
    end
  end

  def install
    bin.install Dir["bin/*"]
    lib.install "lib/dc-common.sh"
    pkgshare.install "config/override.json" if File.exist?("config/override.json")
  end

  test do
    assert_match "dc-tui", shell_output("#{bin}/dc-tui --help")
    assert_match "dc-up", shell_output("#{bin}/dc-up --help")
  end

  def caveats
    <<~EOS
      Needs Docker (Colima or Desktop). Official CLI is separate:
        npm i -g @devcontainers/cli

      Port override example (dc-up --ports):
        #{pkgshare}/override.json
      Copy to ~/.config/devcontainer/override.json if you want it.

      One human, one Docker context. Fleet and prune see the whole engine.
    EOS
  end
end
