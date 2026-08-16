# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers around @devcontainers/cli"
  homepage "https://dc.brasth.com"
  version "0.10.1"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.1/dc-cli-0.10.1-darwin-arm64.tar.gz"
      sha256 "5eacece35598d0dd7bfa95ff237763c10c29dd46476d1bab9fed5f7b3c2091bb"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.1/dc-cli-0.10.1-darwin-amd64.tar.gz"
      sha256 "f4250f86c2aa8f281f62f19c39fc5059514f83b6c3caffecbb6b5deb528c7c54"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.1/dc-cli-0.10.1-linux-arm64.tar.gz"
      sha256 "3caf5a55067e4359bebe8eabd9d9894451b7e552fe004346b06a6ef6a937e26e"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.10.1/dc-cli-0.10.1-linux-amd64.tar.gz"
      sha256 "566cbe3b78f7bb327204d2cecf917b9e4a078775b173eaa047ba72796b8b4118"
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
