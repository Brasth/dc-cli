# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/dc-common.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers around @devcontainers/cli"
  homepage "https://dc.brasth.com"
  version "0.7.2"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.7.2/dc-cli-0.7.2-darwin-arm64.tar.gz"
      sha256 "006ebdd8ef7178cced275606ff8093394adba71b8986bb80a5106be7dbe3ee49"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.7.2/dc-cli-0.7.2-darwin-amd64.tar.gz"
      sha256 "126fc7a28759805caed73baba14dade3b56b02c2c130a8f68ccf97491c3c4ca5"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.7.2/dc-cli-0.7.2-linux-arm64.tar.gz"
      sha256 "c947412d2a05256231d639768f4e56fb6428cbdb40f7473b25564b87f97187ce"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.7.2/dc-cli-0.7.2-linux-amd64.tar.gz"
      sha256 "b247f65c2d1e3b6a98475dcdad1e28cb26ae9f089b9250360882b0dcae6b2c31"
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
