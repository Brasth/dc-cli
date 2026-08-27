# Source of truth for Brasth/homebrew-dc-cli (Formula/dc-cli.rb).
# After a v* release: scripts/print-release-shas.sh vX.Y.Z then update version + sha256.
# Layout: bin/* + lib/*.sh so dirname $0/../lib resolves.
# Do not symlink from libexec — that breaks the scripts' relative lib path.
class DcCli < Formula
  desc "Host-global helpers for Dev Containers and this-folder compose"
  homepage "https://dc.brasth.com"
  version "0.19.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.19.0/dc-cli-0.19.0-darwin-arm64.tar.gz"
      sha256 "be9ed2652310782fe7acaf450b73cd2c105c0a4a703773b6728fe0e3b2b92674"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.19.0/dc-cli-0.19.0-darwin-amd64.tar.gz"
      sha256 "4a72495f32ecb0f9e59108e582045a7e8589e5830fa19f5999fc83574fb9b9b1"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.19.0/dc-cli-0.19.0-linux-arm64.tar.gz"
      sha256 "cd293e5ff1449e1a2977665aeeb6ffe34f22618a00e43722e0a91a0844d163bc"
    end
    on_intel do
      url "https://github.com/Brasth/dc-cli/releases/download/v0.19.0/dc-cli-0.19.0-linux-amd64.tar.gz"
      sha256 "29e25738779e79c50c6ed41585045037091a77ce8c2aba24acb3d36b43d32710"
    end
  end

  def install
    bin.install Dir["bin/*"]
    lib.install Dir["lib/*.sh"]
    pkgshare.install "config/override.json" if File.exist?("config/override.json")
  end

  test do
    assert_match "dc up", shell_output("#{bin}/dc --help")
    assert_match "dc-tui", shell_output("#{bin}/dc-tui --help")
    assert_match "dc-up", shell_output("#{bin}/dc-up --help")
    assert_match "dc-doctor", shell_output("#{bin}/dc-doctor --help")
    assert_match "dc-stats", shell_output("#{bin}/dc-stats --help")
    assert_match "dc-net", shell_output("#{bin}/dc-net --help")
    assert_match "dc-engine", shell_output("#{bin}/dc-engine --help")
    assert_match "dc-try", shell_output("#{bin}/dc-try --help")
  end

  def caveats
    <<~EOS
      Needs Docker (Colima or Desktop — one live engine). Official CLI is required only for
      Dev Container folders. Compose-only folders use docker compose or docker-compose via dc-up.
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
