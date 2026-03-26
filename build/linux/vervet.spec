Name:           vervet
Version:        VERSION_PLACEHOLDER
Release:        1%{?dist}
Summary:        A desktop MongoDB explorer
License:        MIT
URL:            https://github.com/blacktau/vervet
Source0:        https://github.com/blacktau/vervet/releases/download/v%{version}/Vervet-linux-amd64.tar.gz

BuildArch:      x86_64
Requires:       webkit2gtk4.0
Requires:       gtk3

# Binary is pre-built, skip debug package and build ID requirements
%global debug_package %{nil}
%define _build_id_links none

%description
Vervet is a desktop MongoDB explorer built with Go and Vue 3.

%prep
%setup -c -T
cp %{SOURCE0} .
tar xzf Vervet-linux-amd64.tar.gz

%install
install -D -m 755 Vervet %{buildroot}%{_bindir}/Vervet
install -D -m 644 Vervet.desktop %{buildroot}%{_datadir}/applications/Vervet.desktop
install -D -m 644 vervet.png %{buildroot}%{_datadir}/icons/hicolor/512x512/apps/vervet.png

%files
%{_bindir}/Vervet
%{_datadir}/applications/Vervet.desktop
%{_datadir}/icons/hicolor/512x512/apps/vervet.png
