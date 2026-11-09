Name:           vervet
Version:        VERSION_PLACEHOLDER
Release:        1%{?dist}
Summary:        A desktop MongoDB explorer
License:        GPL-3.0-only
URL:            https://github.com/blacktau/vervet
Source0:        %{name}-%{version}.tar.gz

BuildArch:      x86_64
Requires:       webkitgtk6.0
Requires:       gtk4

# Binary is pre-built, skip debug package and build ID requirements
%global debug_package %{nil}
%define _build_id_links none

%description
Vervet is a desktop MongoDB explorer built with Go and Vue 3.

%prep
%setup -q

%install
install -D -m 755 Vervet %{buildroot}%{_bindir}/Vervet
install -D -m 644 com.blacktau.vervet.desktop %{buildroot}%{_datadir}/applications/com.blacktau.vervet.desktop
install -D -m 644 Vervet.png %{buildroot}%{_datadir}/icons/hicolor/512x512/apps/Vervet.png

%files
%{_bindir}/Vervet
%{_datadir}/applications/com.blacktau.vervet.desktop
%{_datadir}/icons/hicolor/512x512/apps/Vervet.png
