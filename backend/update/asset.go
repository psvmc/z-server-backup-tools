package update

import (
	"os"
	"runtime"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

func PreferredAssetName(platform, arch string) string {
	switch platform {
	case "windows":
		if arch == "386" {
			return ProductAssetName + "-386-installer.exe"
		}
		if arch == "arm64" {
			return ProductAssetName + "-arm64-installer.exe"
		}
		return ProductAssetName + "-amd64-installer.exe"
	case "darwin":
		return ProductAssetName + "-macos-universal.zip"
	case "linux":
		return linuxPackageAssetName()
	default:
		return ""
	}
}

func linuxPackageAssetName() string {
	if preferRPM() {
		return ProductAssetName + ".rpm"
	}
	return ProductAssetName + ".deb"
}

func preferRPM() bool {
	for _, path := range []string{
		"/etc/redhat-release",
		"/etc/fedora-release",
		"/etc/SuSE-release",
		"/etc/centos-release",
	} {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return false
	}
	content := strings.ToLower(string(data))
	for _, id := range []string{
		"fedora", "rhel", "centos", "rocky", "almalinux", "opensuse", "suse",
	} {
		if strings.Contains(content, "id="+id) || strings.Contains(content, "id_like="+id) {
			return true
		}
	}
	return false
}

func AssetMatcher(req updater.CheckRequest, assets []github.ReleaseAsset) int {
	platform := req.Platform
	if platform == "" {
		platform = runtime.GOOS
	}
	arch := req.Arch
	if arch == "" {
		arch = runtime.GOARCH
	}

	preferred := PreferredAssetName(platform, arch)
	for i, asset := range assets {
		if asset.Name == preferred {
			return i
		}
	}
	return -1
}
