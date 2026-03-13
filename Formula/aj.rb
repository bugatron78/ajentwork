class Aj < Formula
  desc "Local-first work tracker for AI agents"
  homepage "https://github.com/bugatron78/ajentwork"
  url "https://github.com/bugatron78/ajentwork/archive/refs/tags/v0.1.1.tar.gz"
  sha256 "411e6027536995de3faa698ddeb3a4f86f4c6e729b019193612ab754ad7f32d7"
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
