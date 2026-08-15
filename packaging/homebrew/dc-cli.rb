# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/dc-common.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers around @devcontainers/cli"
  homepage "https://dc.brasth.com"
  version "0.6.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.6.0/dc-cli-0.6.0-darwin-arm64.tar.gz"
      sha256 "0053862c050965a28e3eb0fe8042d440e352a2350fd42b39737ee9b7e60ef467"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.6.0/dc-cli-0.6.0-darwin-amd64.tar.gz"
      sha256 "5d4710ba48cb57a0eb0791049154d18c880d1a42abef358d18b12d02db69eceb"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.6.0/dc-cli-0.6.0-linux-arm64.tar.gz"
      sha256 "73a86183d87c17999c7c1b0289f9de295e885266a5978bde4ad8a978a30903f1"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.6.0/dc-cli-0.6.0-linux-amd64.tar.gz"
      sha256 "3080ab09709b1881dfe212fe399e8a9a71f211c442c118e32e96a2efa9dc6380"
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
