class Snagify < Formula
  desc "CLI to debug works-on-my-machine issues by detecting environment drift"
  homepage "https://github.com/HarshDevelops/snagify"
  version "0.5.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/HarshDevelops/snagify/releases/download/v0.5.0/snagify_Darwin_arm64.tar.gz"
      sha256 "1c395a9fde54711c6910f34345928162d0088f73b4e62f7b5cbce0c40973fb47"
    else
      url "https://github.com/HarshDevelops/snagify/releases/download/v0.5.0/snagify_Darwin_x86_64.tar.gz"
      sha256 "50d7fc4068749ce69f1042542ce799e342172bdb3eedeecef281d57b5e41498f"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/HarshDevelops/snagify/releases/download/v0.5.0/snagify_Linux_arm64.tar.gz"
      sha256 "5cc1a0ebfc75370e3e1a985d3e99f2197db51c4057cf98ae85b536db81610a52"
    else
      url "https://github.com/HarshDevelops/snagify/releases/download/v0.5.0/snagify_Linux_x86_64.tar.gz"
      sha256 "5a9fc0747f24490807902fc6c25c9336704f2dac2f4e8fa433e6ce7c8c03e564"
    end
  end

  def install
    bin.install "snagify"
  end

  test do
    system "#{bin}/snagify", "--version"
  end
end
