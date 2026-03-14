class Aj < Formula
  desc "Local-first work tracker for AI agents"
  homepage "https://github.com/bugatron78/ajentwork"
  url "https://github.com/bugatron78/ajentwork/archive/refs/tags/v0.1.4.tar.gz"
  sha256 "6fad9c74696731d82b6e58a89503005885edc0d4c59ec282fa3a40276f6198fb"
  license "Apache-2.0"

  depends_on "go" => :build

  def install
    version_flag = "-X ajentwork/internal/buildinfo.Version=v#{version}"
    system "go", "build", *std_go_args(ldflags: "-s -w #{version_flag}"), "./cmd/aj"
    man1.install "docs/aj.1"
  end

  test do
    output = shell_output("#{bin}/aj --version")
    assert_match version.to_s, output
  end
end
