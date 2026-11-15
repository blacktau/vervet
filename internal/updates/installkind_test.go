package updates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectInstall(t *testing.T) {
	cases := []struct {
		name      string
		goos      string
		exe       string
		appImage  string
		writable  bool
		wantKind  InstallKind
		wantTgt   string
		wantSelfU bool
	}{
		{"appimage", "linux", "/tmp/.mount_VervetX/usr/bin/Vervet", "/home/u/Apps/Vervet.AppImage", true, InstallAppImage, "/home/u/Apps/Vervet.AppImage", true},
		{"appimage in read-only dir", "linux", "/tmp/.mount_VervetX/usr/bin/Vervet", "/opt/Vervet.AppImage", false, InstallAppImage, "/opt/Vervet.AppImage", false},
		{"rpm", "linux", "/usr/bin/vervet", "", false, InstallPackageManager, "/usr/bin/vervet", false},
		{"rpm even if writable", "linux", "/usr/bin/vervet", "", true, InstallPackageManager, "/usr/bin/vervet", false},
		{"bare linux binary", "linux", "/home/u/src/vervet/bin/Vervet", "", true, InstallOther, "/home/u/src/vervet/bin/Vervet", false},
		{"windows user install", "windows", `C:\Users\u\AppData\Local\Programs\Vervet\Vervet.exe`, "", true, InstallWindows, `C:\Users\u\AppData\Local\Programs\Vervet\Vervet.exe`, true},
		{"windows all-users install", "windows", `C:\Program Files\Blacktau\Vervet\Vervet.exe`, "", false, InstallWindows, `C:\Program Files\Blacktau\Vervet\Vervet.exe`, false},
		{"mac app", "darwin", "/Applications/Vervet.app/Contents/MacOS/Vervet", "", true, InstallMacApp, "/Applications/Vervet.app", true},
		{"mac app not writable", "darwin", "/Applications/Vervet.app/Contents/MacOS/Vervet", "", false, InstallMacApp, "/Applications/Vervet.app", false},
		{"bare mac binary", "darwin", "/Users/u/bin/Vervet", "", true, InstallOther, "/Users/u/bin/Vervet", false},
		{"APPIMAGE ignored off linux", "darwin", "/Applications/Vervet.app/Contents/MacOS/Vervet", "/x.AppImage", true, InstallMacApp, "/Applications/Vervet.app", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := detectInstall(installEnv{
				goos: c.goos,
				exe:  c.exe,
				getenv: func(k string) string {
					if k == "APPIMAGE" {
						return c.appImage
					}
					return ""
				},
				writable: func(string) bool { return c.writable },
			})
			if got.Kind != c.wantKind || got.Target != c.wantTgt || got.CanSelfUpdate != c.wantSelfU {
				t.Errorf("got %+v, want kind=%s target=%s self=%v", got, c.wantKind, c.wantTgt, c.wantSelfU)
			}
		})
	}
}

func TestDirWritable(t *testing.T) {
	dir := t.TempDir()
	if !dirWritable(dir) {
		t.Error("temp dir not writable")
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("probe file left behind: %v", entries)
	}
	if dirWritable(filepath.Join(dir, "missing")) {
		t.Error("missing dir reported writable")
	}
}
