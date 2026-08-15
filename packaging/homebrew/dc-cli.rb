# Template for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: fill version + sha256 from the four tarballs.
# Layout: bin/* + lib/dc-common.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers around @devcontainers/cli"
  homepage "https://dc.brasth.com"
  version "REPLACE_VERSION"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/vREPLACE_VERSION/dc-cli-REPLACE_VERSION-darwin-arm64.tar.gz"
      sha256 "REPLACE_SHA256_DARWIN_ARM64"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/vREPLACE_VERSION/dc-cli-REPLACE_VERSION-darwin-amd64.tar.gz"
      sha256 "REPLACE_SHA256_DARWIN_AMD64"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/vREPLACE_VERSION/dc-cli-REPLACE_VERSION-linux-arm64.tar.gz"
      sha256 "REPLACE_SHA256_LINUX_ARM64"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/vREPLACE_VERSION/dc-cli-REPLACE_VERSION-linux-amd64.tar.gz"
      sha256 "REPLACE_SHA256_LINUX_AMD64"
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
