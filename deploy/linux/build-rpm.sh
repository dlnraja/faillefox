#!/usr/bin/env bash
# Build un paquet .rpm pour Faillefox (Fedora / RHEL / openSUSE).
#
# Usage (depuis la racine du dépôt, sur un système Linux avec rpmbuild) :
#   ./deploy/linux/build-rpm.sh [version]
#
# Produit : dist/rpmbuild/RPMS/x86_64/faillefox-<version>-1.x86_64.rpm
set -euo pipefail
cd "$(dirname "$0")/../.."
VERSION="${1:-$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo 0.0.0)}"

echo "==> Build binaire Linux amd64 (v${VERSION})"
mkdir -p /tmp/faillefox-src-${VERSION}
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags="-s -w" -o /tmp/faillefox-src-${VERSION}/faillefox ./cmd/faillefox
# Copie des fichiers requis par le .spec dans le tarball source.
cp -r deploy LICENSE README.md /tmp/faillefox-src-${VERSION}/

echo "==> Préparation du tarball source"
mkdir -p dist
tar czf "dist/${PWD##*/}-${VERSION}.tar.gz" -C /tmp faillefox-src-${VERSION}
mv "dist/faillefox-${VERSION}.tar.gz" dist/ 2>/dev/null || true
# rpmbuild attend %{name}-%{version}.tar.gz (= faillefox-<version>.tar.gz).
ln -sf "${PWD##*/}-${VERSION}.tar.gz" "dist/faillefox-${VERSION}.tar.gz"

echo "==> rpmbuild"
TOPDIR="$(pwd)/dist/rpmbuild"
mkdir -p "${TOPDIR}/"{BUILD,RPMS,SOURCES,SPECS,SRPMS}
cp deploy/linux/faillefox.spec "${TOPDIR}/SPECS/"
cp "dist/faillefox-${VERSION}.tar.gz" "${TOPDIR}/SOURCES/"

rpmbuild -bb \
  --define "_topdir ${TOPDIR}" \
  --define "version ${VERSION}" \
  "${TOPDIR}/SPECS/faillefox.spec"

echo "==> RPM produit :"
find "${TOPDIR}/RPMS" -name "faillefox-*.rpm" -exec ls -la {} \;
rm -rf /tmp/faillefox-src-${VERSION}
