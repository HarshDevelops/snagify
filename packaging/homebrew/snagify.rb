class Snagify < Formula
  desc "CLI to debug works-on-my-machine issues by detecting environment drift"
  homepage "https://github.com/HarshDevelops/snagify"
  version "0.4.1"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/HarshDevelops/snagify/releases/download/v0.4.1/snagify_Darwin_arm64.tar.gz"
      sha256 "232f7db92df1c563565fe4da0688e54333fc0ae7d0952a0b2630ae485e3d058e"
    else
      url "https://github.com/HarshDevelops/snagify/releases/download/v0.4.1/snagify_Darwin_x86_64.tar.gz"
      sha256 "c5f83cca5323a539896694c331cf36aa6511e30c7c2eaa732c39358c89cfb670"
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/HarshDevelops/snagify/releases/download/v0.4.1/snagify_Linux_arm64.tar.gz"
      sha256 "fb0ed6c2222bc8f6c685fb3ca2199932faa59f9b80051c2488f2ccbf207c138b"
    else
      url "https://github.com/HarshDevelops/snagify/releases/download/v0.4.1/snagify_Linux_x86_64.tar.gz"
      sha256 "f273a4d235f4dda7319aca18ba3fa72444c98dc44dca5971620197a0f84ab5bc"
    end
  end

  def install
    bin.install "snagify"
  end

  test do
    system "#{bin}/snagify", "--version"
  end
end
