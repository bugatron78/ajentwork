class Aj < Formula
  desc "Local-first work tracker for AI agents"
  homepage "https://github.com/bugatron78/ajentwork"
  url "https://github.com/bugatron78/ajentwork/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "280f81970d3451de693cb4e037fc0e64fd124cbbf7f754efde0bc99ccd39e43a"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/aj"
  end

  test do
    output = shell_output("#{bin}/aj --help")
    assert_match "agent work tracker", output
  end
end
