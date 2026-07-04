class Promptq < Formula
  desc "Fast local prompt manager for terminal and TUI workflows"
  homepage "https://github.com/ast-lw/promptq"
  license "MIT"

  head "https://github.com/ast-lw/promptq.git", branch: "main"

  depends_on "go" => :build

  def install
    ldflags = "-s -w -X github.com/ast-lw/promptq/internal/promptq.Version=#{version}"
    system "go", "build", "-trimpath", "-ldflags", ldflags, "-o", bin/"promptq", "./cmd/promptq"
  end

  test do
    assert_match "promptq", shell_output("#{bin}/promptq version")
  end
end
