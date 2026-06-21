Name:       faillefox
Version:    0.7.1
Release:    1%{?dist}
Summary:    Pare-feu libre et multiplateforme (protection réseau, DNS sinkhole, veille CVE, scanner ClamAV)

License:    GPL-3.0-only
URL:        https://github.com/dlnraja/faillefox
Source0:    %{name}-%{version}.tar.gz

# Faillefox est en Go pur (CGO_ENABLED=0) : aucune dépendance runtime autre que nftables.
Requires:   nftables
Requires(pre): shadow-utils

%description
Faillefox intercepte le trafic réseau sortant et laisse l'utilisateur décider,
par application, ce qui a le droit de sortir sur Internet. Inclut un résolveur
DNS sinkhole (anti-pubs/trackers/malwares), une veille CVE (base NVD), un
scanner ClamAV, un agrégateur de threat intelligence (Abuse.ch, OTX) et un
corrélateur d'alertes.

%global debug_package %{nil}

%prep
%setup -q

%build
# Le binaire est déjà compilé dans le tarball (cross-compile depuis Go).
# Pas de build à faire côté RPM (CGO désactivé).

%install
install -D -m 0755 faillefox %{buildroot}%{_bindir}/faillefox
install -D -m 0644 deploy/linux/faillefox.service %{buildroot}%{_unitdir}/faillefox.service
install -D -m 0644 LICENSE %{buildroot}%{_docdir}/%{name}/LICENSE
install -D -m 0644 README.md %{buildroot}%{_docdir}/%{name}/README.md
install -d -m 0755 %{buildroot}%{_sharedstatedir}/%{name}

%pre
# Crée un utilisateur système dédié (Faillefox n'a pas besoin de compte interactif).
getent group faillefox >/dev/null || groupadd -r faillefox
getent passwd faillefox >/dev/null || useradd -r -g faillefox -s /sbin/nologin \
    -d %{_sharedstatedir}/%{name} faillefox

%post
%systemd_post faillefox.service

%preun
%systemd_preun faillefox.service

%postun
%systemd_postun faillefox.service

%files
%license LICENSE
%doc README.md
%{_bindir}/faillefox
%{_unitdir}/faillefox.service
%attr(0755,faillefox,faillefox) %{_sharedstatedir}/%{name}

%changelog
* Sat Jun 21 2026 dlnraja <dylan.rajasekaram@gmail.com> - 0.7.1-1
- Packaging RPM initial : binaire + service systemd + utilisateur dédié.
