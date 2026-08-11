package remote

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"z-server-backup-tools/backend/model"
	"z-server-backup-tools/backend/util"
)

// BuildRemoteTempZipPath returns a remote temp zip path for server-side compression.
// On Windows the caller should use Client.ResolveRemoteTempZipPath to expand %TEMP%.
func BuildRemoteTempZipPath(osType, zipBaseName, token string) string {
	osType = model.NormalizeOSType(osType)
	name := strings.TrimSpace(zipBaseName)
	if name == "" {
		name = "backup"
	}
	name = strings.TrimSuffix(name, ".zip") + ".zip"
	file := fmt.Sprintf("z-srv-backup-%s-%s", strings.TrimSpace(token), name)
	if model.IsLinuxOS(osType) {
		return util.JoinRemoteForOS(osType, "/tmp", file)
	}
	return `%TEMP%\` + file
}

func tempZipFileName(zipBaseName, token string) string {
	name := strings.TrimSpace(zipBaseName)
	if name == "" {
		name = "backup"
	}
	name = strings.TrimSuffix(name, ".zip")
	return fmt.Sprintf("z-srv-backup-%s-%s.zip", strings.TrimSpace(token), name)
}

func tempListFileName(token string) string {
	return fmt.Sprintf("z-srv-backup-%s.list", strings.TrimSpace(token))
}

// ResolveRemoteTempListPath returns an absolute remote temp list file path.
func (c *Client) ResolveRemoteTempListPath(token string) (string, error) {
	osType := model.NormalizeOSType(c.cfg.OSType)
	file := tempListFileName(token)
	if model.IsLinuxOS(osType) {
		return util.JoinRemoteForOS(osType, "/tmp", file), nil
	}
	temp, err := c.RunShell("echo %TEMP%")
	if err != nil {
		return "", fmt.Errorf("读取远程 TEMP 目录失败: %w", err)
	}
	temp = strings.TrimSpace(temp)
	if temp == "" {
		return "", fmt.Errorf("远程 TEMP 目录为空")
	}
	return util.JoinRemoteForOS(osType, temp, file), nil
}

// ResolveRemoteTempZipPath returns an absolute remote temp zip path (expands %TEMP% on Windows).
func (c *Client) ResolveRemoteTempZipPath(zipBaseName, token string) (string, error) {
	osType := model.NormalizeOSType(c.cfg.OSType)
	file := tempZipFileName(zipBaseName, token)
	if model.IsLinuxOS(osType) {
		return util.JoinRemoteForOS(osType, "/tmp", file), nil
	}
	temp, err := c.RunShell("echo %TEMP%")
	if err != nil {
		return "", fmt.Errorf("读取远程 TEMP 目录失败: %w", err)
	}
	temp = strings.TrimSpace(temp)
	if temp == "" {
		return "", fmt.Errorf("远程 TEMP 目录为空")
	}
	return util.JoinRemoteForOS(osType, temp, file), nil
}

func remoteFolderParts(remoteFolder, osType string) (parent, base string, err error) {
	remoteFolder = util.NormalizeRemotePathForOS(remoteFolder, osType)
	if remoteFolder == "" {
		return "", "", fmt.Errorf("远程文件夹路径不能为空")
	}
	if model.IsLinuxOS(osType) {
		base = path.Base(remoteFolder)
		parent = path.Dir(remoteFolder)
	} else {
		base = filepath.Base(remoteFolder)
		parent = filepath.Dir(remoteFolder)
	}
	if base == "." || base == "/" || base == `\` || base == "" {
		return "", "", fmt.Errorf("无法从远程路径解析文件夹名")
	}
	return parent, base, nil
}

func linuxZipCheckCmd() string {
	return "command -v zip >/dev/null 2>&1"
}

func linuxZipInstallCmd() string {
	return `command -v zip >/dev/null 2>&1 || (
if command -v apt-get >/dev/null 2>&1; then sudo -n apt-get update -qq && sudo -n apt-get install -y zip;
elif command -v yum >/dev/null 2>&1; then sudo -n yum install -y zip;
elif command -v dnf >/dev/null 2>&1; then sudo -n dnf install -y zip;
elif command -v apk >/dev/null 2>&1; then sudo -n apk add --no-cache zip;
else exit 127;
fi
)`
}

func buildLinuxCompressCmd(remoteFolder, remoteZip string) (string, error) {
	parent, base, err := remoteFolderParts(remoteFolder, model.OSTypeLinux)
	if err != nil {
		return "", err
	}
	remoteZip = util.NormalizeRemotePathForOS(remoteZip, model.OSTypeLinux)
	return fmt.Sprintf(
		"cd %s && zip -r -q %s %s",
		util.QuoteShellArg(parent),
		util.QuoteShellArg(remoteZip),
		util.QuoteShellArg(base),
	), nil
}

func buildWindowsCompressCmd(remoteFolder, remoteZip string) (string, error) {
	parent, base, err := remoteFolderParts(remoteFolder, model.OSTypeWindows)
	if err != nil {
		return "", err
	}
	remoteFolder = util.NormalizeRemotePathForOS(remoteFolder, model.OSTypeWindows)
	remoteZip = util.NormalizeRemotePathForOS(remoteZip, model.OSTypeWindows)
	// tar -a creates zip; -C parent + folder name keeps the folder layer in archive.
	return fmt.Sprintf(
		`tar -a -c -f %s -C %s %s`,
		util.QuoteWindowsArg(remoteZip),
		util.QuoteWindowsArg(parent),
		util.QuoteWindowsArg(base),
	), nil
}

func buildWindowsCompressPSCmd(remoteFolder, remoteZip string) (string, error) {
	remoteFolder = util.NormalizeRemotePathForOS(remoteFolder, model.OSTypeWindows)
	remoteZip = util.NormalizeRemotePathForOS(remoteZip, model.OSTypeWindows)
	// Compress-Archive includes the selected folder name as top-level entries.
	inner := fmt.Sprintf(
		"Compress-Archive -LiteralPath %s -DestinationPath %s -Force",
		util.QuoteShellArg(remoteFolder),
		util.QuoteShellArg(remoteZip),
	)
	return "powershell -NoProfile -Command " + util.QuoteWindowsArg(inner), nil
}

func buildRemoteRemoveCmd(remotePath, osType string) string {
	remotePath = util.NormalizeRemotePathForOS(remotePath, osType)
	if model.IsLinuxOS(osType) {
		return fmt.Sprintf("rm -f %s", util.QuoteShellArg(remotePath))
	}
	return fmt.Sprintf("del /f /q %s", util.QuoteWindowsArg(remotePath))
}

// EnsureZipTool checks zip capability on the server; on Linux may try non-interactive install.
func (c *Client) EnsureZipTool(logf func(string)) error {
	osType := model.NormalizeOSType(c.cfg.OSType)
	if !model.IsLinuxOS(osType) {
		if logf != nil {
			logf("Windows 服务器使用内置 tar/PowerShell 压缩，无需安装 zip")
		}
		return nil
	}
	if _, err := c.RunShell(linuxZipCheckCmd()); err == nil {
		if logf != nil {
			logf("远程已安装 zip 命令")
		}
		return nil
	}
	if logf != nil {
		logf("远程未检测到 zip，尝试自动安装（需要免密 sudo）…")
	}
	if _, err := c.RunShell(linuxZipInstallCmd()); err != nil {
		return fmt.Errorf("zip 自动安装失败: %w", err)
	}
	if _, err := c.RunShell(linuxZipCheckCmd()); err != nil {
		return fmt.Errorf("zip 安装后仍不可用")
	}
	if logf != nil {
		logf("zip 安装成功")
	}
	return nil
}

func buildLinuxCompressListCmd(parent, remoteZip, listFile string) (string, error) {
	parent = util.NormalizeRemotePathForOS(parent, model.OSTypeLinux)
	remoteZip = util.NormalizeRemotePathForOS(remoteZip, model.OSTypeLinux)
	listFile = util.NormalizeRemotePathForOS(listFile, model.OSTypeLinux)
	return fmt.Sprintf(
		"cd %s && zip -q %s -@ < %s",
		util.QuoteShellArg(parent),
		util.QuoteShellArg(remoteZip),
		util.QuoteShellArg(listFile),
	), nil
}

func buildWindowsCompressListCmd(parent, remoteZip, listFile string) (string, error) {
	parent = util.NormalizeRemotePathForOS(parent, model.OSTypeWindows)
	remoteZip = util.NormalizeRemotePathForOS(remoteZip, model.OSTypeWindows)
	listFile = util.NormalizeRemotePathForOS(listFile, model.OSTypeWindows)
	return fmt.Sprintf(
		`tar -a -c -f %s -C %s -T %s`,
		util.QuoteWindowsArg(remoteZip),
		util.QuoteWindowsArg(parent),
		util.QuoteWindowsArg(listFile),
	), nil
}

// CompressFolderRemote runs server-side compression; archive includes the folder name layer.
func (c *Client) CompressFolderRemote(remoteFolder, remoteZip string) error {
	osType := model.NormalizeOSType(c.cfg.OSType)
	if model.IsLinuxOS(osType) {
		cmd, err := buildLinuxCompressCmd(remoteFolder, remoteZip)
		if err != nil {
			return err
		}
		_, err = c.RunShell(cmd)
		return err
	}
	cmd, err := buildWindowsCompressCmd(remoteFolder, remoteZip)
	if err != nil {
		return err
	}
	if _, err := c.RunShell(cmd); err == nil {
		return nil
	}
	psCmd, err := buildWindowsCompressPSCmd(remoteFolder, remoteZip)
	if err != nil {
		return err
	}
	_, err = c.RunShell(psCmd)
	return err
}

// CompressFolderRemoteWithList compresses only paths listed in listFile (relative to parent of remoteFolder).
func (c *Client) CompressFolderRemoteWithList(remoteFolder, remoteZip, listFile string) error {
	osType := model.NormalizeOSType(c.cfg.OSType)
	parent, _, err := remoteFolderParts(remoteFolder, osType)
	if err != nil {
		return err
	}
	if model.IsLinuxOS(osType) {
		cmd, err := buildLinuxCompressListCmd(parent, remoteZip, listFile)
		if err != nil {
			return err
		}
		_, err = c.RunShell(cmd)
		return err
	}
	cmd, err := buildWindowsCompressListCmd(parent, remoteZip, listFile)
	if err != nil {
		return err
	}
	_, err = c.RunShell(cmd)
	return err
}

// RemoveRemoteFile deletes a remote file via SSH shell.
func (c *Client) RemoveRemoteFile(remotePath string) error {
	_, err := c.RunShell(buildRemoteRemoveCmd(remotePath, c.cfg.OSType))
	return err
}
