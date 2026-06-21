# Formule Homebrew pour Faillefox.
# -----------------------------------------------------------------------------
# Permet l'installation en une commande sur macOS (et Linux via Linuxbrew) :
#
#   brew tap dlnraja/faillefox https://github.com/dlnraja/faillefox
#   brew install faillefox
#   brew services start faillefox     # démarre au boot via launchd
#
# Cette formule sera publiée dans un tap dédié (github.com/dlnraja/homebrew-faillefox)
# une fois le dépôt stable. En attendant, elle peut être utilisée localement :
#
#   brew install --HEAD ./deploy/macos/faillefox.rb
# -----------------------------------------------------------------------------
class Faillefox < Formula
  desc "Pare-feu libre et multiplateforme (DNS sinkhole, CVE, ClamAV, threat intel)"
  homepage "https://github.com/dlnraja/faillefox"
  url "https://github.com/dlnraja/faillefox/archive/refs/tags/v0.7.1.tar.gz"
  sha256 "REPLACED_BY_RELEASE_WORKFLOW"
  version "0.7.1"
  license "GPL-3.0"

  # Faillefox est en Go pur (CGO_ENABLED=0) : aucune dépendance binaire.
  depends_on "go" => :build

  def install
    # Build du binaire pour l'OS courant (macOS Intel/ARM, ou Linux).
    system "go", "build", *std_go_args(ldflags: "-s -w -trimpath",
                                       output: bin/"faillefox"),
           "./cmd/faillefox"
    # Page de manuel (optionnelle, si générée plus tard).
  end

  # Au démarrage via `brew services`, on active le DNS sinkhole + CVE.
  service do
    run [opt_bin/"faillefox", "-dns", "-cve", "-threat-intel"]
    keep_alive true
    run_at_load true
    log_path var/"log/faillefox.log"
    error_log_path var/"log/faillefox.err.log"
  end

  test do
    # `faillefox -list-drivers` doit lister au moins le stub.
    output = shell_output("#{bin}/faillefox -list-drivers")
    assert_match "stub", output
  end
end
