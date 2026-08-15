# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/dc-common.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers around @devcontainers/cli"
  homepage "https://dc.brasth.com"
  version "0.7.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.7.0/dc-cli-0.7.0-darwin-arm64.tar.gz"
      sha256 "b6f50e5685fe98848ce7c0b5e623efdf9854487fbeade05e52e6f5fe0cc4b690"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.7.0/dc-cli-0.7.0-darwin-amd64.tar.gz"
      sha256 "f243e654d347e11544d25b78f4288d695335a23b16c95d5edd999cb386269bcd"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.7.0/dc-cli-0.7.0-linux-arm64.tar.gz"
      sha256 "e069ce5a4d8287e98656228506e0a83da380dea4590d902f5dd59fceefef4ecb"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.7.0/dc-cli-0.7.0-linux-amd64.tar.gz"
      sha256 "6fdecc725bda6c5f1d9fb3bd5faedce803d5e538b89c1dc41ad90a6d9fe8fb29"
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
