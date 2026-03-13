class Aj < Formula
  desc "Local-first work tracker for AI agents"
  homepage "https://github.com/bugatron78/ajentwork"
  url "https://github.com/bugatron78/ajentwork/archive/refs/tags/v0.1.2.tar.gz"
  sha256 "6159d5363daa4f3dbc14782743dd69148538bf4f2759bdab8ccb3a1cd5e061b7"
  license "Apache-2.0"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/aj"
    man1.install "docs/aj.1"
  end

  test do
    output = shell_output("#{bin}/aj --help")
    assert_match "agent work tracker", output
  end
end
