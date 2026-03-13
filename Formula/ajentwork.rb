class Ajentwork < Formula
  desc "Local-first work tracker for AI agents"
  homepage "https://github.com/bugatron78/ajentwork"
  version "0.1.0"

  if OS.mac? && Hardware::CPU.intel?
    url "https://github.com/bugatron78/ajentwork/releases/download/v0.1.0/aj_v0.1.0_darwin_amd64.tar.gz"
    sha256 "2d5f0356a27e096339d8f99ad8653a315d0a5c0b04d0f60ae6ccc7230e0707de"
  elsif OS.mac? && Hardware::CPU.arm?
    url "https://github.com/bugatron78/ajentwork/releases/download/v0.1.0/aj_v0.1.0_darwin_arm64.tar.gz"
    sha256 "e67a44cf3473e0e01011b8f8734b46a2dc5459950746419331a71d44af874961"
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/bugatron78/ajentwork/releases/download/v0.1.0/aj_v0.1.0_linux_amd64.tar.gz"
    sha256 "0dec18267f9489d750ca203f7ad4433f1557ffb292ae79e5e7301cb4a81e0b7b"
  else
    url "https://github.com/bugatron78/ajentwork/releases/download/v0.1.0/aj_v0.1.0_linux_arm64.tar.gz"
    sha256 "6c26c1940f50c53ec2ba57679d61e05d7bc7437038e98347cccf2d0fb8e0885c"
  end

  def install
    bin.install "aj"
  end

  test do
    output = shell_output("#{bin}/aj --help")
    assert_match "agent work tracker", output
  end
end
