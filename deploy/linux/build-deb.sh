#!/usr/bin/env bash
# Build un paquet .deb pour Faillefox (Linux amd64).
#
# Usage :
#   ./deploy/linux/build-deb.sh [version]
#   (version défaut : extrait du dernier tag git, ex: 0.5.0)
#
# Produit : dist/faillefox_<version>_amd64.deb
#
# Le paquet installe :
#   - /usr/bin/faillefox          (le binaire)
#   - /etc/systemd/system/faillefox.service
#   - /usr/share/doc/faillefox/*  (README, LICENSE)
#   - hook postinst qui active + démarre le service
set -euo pipefail

cd "$(dirname "$0")/../.."
VERSION="${1:-$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo 0.0.0)}"
BUILD_DIR="$(mktemp -d)"
PKG="faillefox_${VERSION}_amd64"
DEST="${BUILD_DIR}/${PKG}"

echo "==> Build binaire Linux amd64 (v${VERSION})"
mkdir -p "${DEST}/usr/bin" "${DEST}/etc/systemd/system" \
         "${DEST}/usr/share/doc/faillefox" "${DEST}/var/lib/faillefox"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags="-s -w" \
  -o "${DEST}/usr/bin/faillefox" ./cmd/faillefox

echo "==> Copie fichiers du paquet"
cp deploy/linux/faillefox.service "${DEST}/etc/systemd/system/"
cp LICENSE README.md CHANGELOG.md "${DEST}/usr/share/doc/faillefox/" 2>/dev/null || true

echo "== control =="
mkdir -p "${DEST}/DEBIAN"
cat > "${DEST}/DEBIAN/control" <<EOF
Package: faillefox
Version: ${VERSION}
Architecture: amd64
Maintainer: dlnraja <dylan.rajasekaram@gmail.com>
Section: net
Priority: optional
Description: Faillefox - pare-feu libre et multiplateforme
 Faillefox intercepte le trafic réseau sortant et vous laisse décider,
 par application, ce qui a le droit de sortir sur Internet. Inclut un
 résolveur DNS sinkhole (anti-pubs/trackers/malwares), une veille CVE
 (base NVD) et un scanner ClamAV.
Homepage: https://github.com/dlnraja/faillefox
Depends: nftables
EOF

echo "== postinst (activation du service) =="
cat > "${DEST}/DEBIAN/postinst" <<'EOF'
#!/bin/sh
set -e
if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload
  systemctl enable faillefox
  systemctl restart faillefox || true
  echo "Service faillefox activé. Statut: systemctl status faillefox"
fi
EOF
chmod 0755 "${DEST}/DEBIAN/postinst"

echo "==> Construction du .deb"
mkdir -p dist
dpkg-deb --build --root-owner-group "${DEST}" "dist/${PKG}.deb"
rm -rf "${BUILD_DIR}"
echo "==> OK: dist/${PKG}.deb"
ls -la "dist/${PKG}.deb"
