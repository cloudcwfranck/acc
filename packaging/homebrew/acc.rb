# Homebrew Formula for acc
#
# This is a DRAFT formula for acc. It is NOT published to Homebrew yet.
#
# To publish this formula to a Homebrew tap:
# 1. Create a tap repository: homebrew-acc
# 2. Copy this file to Formula/acc.rb
# 3. Update the sha256 checksums with actual release values
# 4. Push to GitHub
# 5. Users can install with: brew tap cloudcwfranck/acc && brew install acc
#
# Official Homebrew documentation: https://docs.brew.sh/Formula-Cookbook

class Acc < Formula
  desc "Secure Workload Accelerator - Policy-gated OCI workload verification"
  homepage "https://github.com/cloudcwfranck/acc"
  version "0.1.0"

  # Update these URLs and checksums after v0.1.0 is released
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/cloudcwfranck/acc/releases/download/v#{version}/acc_#{version}_darwin_arm64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_RELEASE"  # Get from checksums.txt
    else
      url "https://github.com/cloudcwfranck/acc/releases/download/v#{version}/acc_#{version}_darwin_amd64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_RELEASE"  # Get from checksums.txt
    end
  elsif OS.linux?
    if Hardware::CPU.arm?
      url "https://github.com/cloudcwfranck/acc/releases/download/v#{version}/acc_#{version}_linux_arm64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_RELEASE"  # Get from checksums.txt
    else
      url "https://github.com/cloudcwfranck/acc/releases/download/v#{version}/acc_#{version}_linux_amd64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256_AFTER_RELEASE"  # Get from checksums.txt
    end
  end

  license "Apache-2.0"

  # Dependencies
  # acc has no runtime dependencies, only optional tools like docker/podman/syft

  def install
    # Determine binary name based on platform
    if OS.mac?
      if Hardware::CPU.arm?
        bin.install "acc-darwin-arm64" => "acc"
      else
        bin.install "acc-darwin-amd64" => "acc"
      end
    elsif OS.linux?
      if Hardware::CPU.arm?
        bin.install "acc-linux-arm64" => "acc"
      else
        bin.install "acc-linux-amd64" => "acc"
      end
    end
  end

  test do
    # Test that the binary runs and shows version
    assert_match version.to_s, shell_output("#{bin}/acc version")
  end
end

# How to update this formula for new releases:
#
# 1. Update the version number (line 15)
# 2. After releasing v0.1.0, download checksums.txt from GitHub Releases:
#    curl -L https://github.com/cloudcwfranck/acc/releases/download/v0.1.0/checksums.txt
# 3. Replace each REPLACE_WITH_ACTUAL_SHA256_AFTER_RELEASE with the corresponding checksum:
#    - darwin_arm64: Get checksum for acc_0.1.0_darwin_arm64.tar.gz
#    - darwin_amd64: Get checksum for acc_0.1.0_darwin_amd64.tar.gz
#    - linux_arm64: Get checksum for acc_0.1.0_linux_arm64.tar.gz
#    - linux_amd64: Get checksum for acc_0.1.0_linux_amd64.tar.gz
# 4. Test locally: brew install --build-from-source ./Formula/acc.rb
# 5. Test formula: brew test acc
# 6. Audit formula: brew audit --strict --online acc
#
# Example after release:
#   sha256 "abc123def456..." (actual checksum from checksums.txt)
