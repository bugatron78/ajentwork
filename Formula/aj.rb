class Aj < Formula
  desc "Local-first work tracker for AI agents"
  homepage "https://github.com/bugatron78/ajentwork"
  url "https://github.com/bugatron78/ajentwork/archive/refs/tags/v0.1.6.tar.gz"
  sha256 "80e50465cd1e7b085306c71f7c5ac5690a9b48f2843b46836ee020e66aa865a7"
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
