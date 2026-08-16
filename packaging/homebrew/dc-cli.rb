# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers around @devcontainers/cli"
  homepage "https://dc.brasth.com"
  version "0.10.2"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.2/dc-cli-0.10.2-darwin-arm64.tar.gz"
      sha256 "226a67a1fa881e203afac85e7388e6215044b28751e42d9d5e5b8902576c0f98"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.2/dc-cli-0.10.2-darwin-amd64.tar.gz"
      sha256 "b1946d36269bef25503f96c9e7baf7e6ca76883b2a567f90222adb6acfb1ebba"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.2/dc-cli-0.10.2-linux-arm64.tar.gz"
      sha256 "3c79577a5f262fae84c71e483d818512e5984b400534c14926568cc7fc65bbb1"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.2/dc-cli-0.10.2-linux-amd64.tar.gz"
      sha256 "7e7fc72b531304cf0793782bc239b4794e2152a86621b24a104ed318dde38994"
    end
  end

  depends_on "yazi" => :recommended

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
