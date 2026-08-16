# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers around @devcontainers/cli"
  homepage "https://dc.brasth.com"
  version "0.10.4"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.4/dc-cli-0.10.4-darwin-arm64.tar.gz"
      sha256 "72fbbd7a007adee69f5cac09e8068d2353c7845770fdcde4b8a692493ef10022"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.4/dc-cli-0.10.4-darwin-amd64.tar.gz"
      sha256 "e962cf451df56c424730bedbe712fa05d1359477bb1e17e4c6f09a68c99a319b"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.4/dc-cli-0.10.4-linux-arm64.tar.gz"
      sha256 "c76eaaf735e75ed2364af9ba7c565cb93a9606d721a408764888e734aa77c3c0"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.4/dc-cli-0.10.4-linux-amd64.tar.gz"
      sha256 "15899064b99d9798e7d607fc8168ef9512e5691f53d77ede51195f1a15cfdb60"
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
