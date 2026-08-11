package sftpclient

import (
	"fmt"
	"path"
	"strings"

	"z-server-backup-tools/backend/model"
	"z-server-backup-tools/backend/util"
)

// ListIncludedRelPaths walks a remote folder via SFTP and returns file paths relative to
// the parent directory (e.g. myapp/logs/app.log), after applying ignore patterns.
// onIgnored is called for each skipped file or directory (entire subtree skipped for dirs).
func (c *Client) ListIncludedRelPaths(remoteDir, osType string, ignorePatterns []string, onIgnored func(string)) ([]string, error) {
	osType = model.NormalizeOSType(osType)
	remoteDir = strings.TrimSpace(remoteDir)
	if remoteDir == "" {
		return nil, fmt.Errorf("远程文件夹路径不能为空")
	}
	folderName, err := remoteFolderBaseName(remoteDir)
	if err != nil {
		return nil, err
	}
	rootSFTP := toSFTPPath(remoteDir, osType)
	info, err := c.sftp.Stat(rootSFTP)
	if err != nil {
		return nil, fmt.Errorf("读取远程文件夹失败: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("远程路径不是文件夹: %s", remoteDir)
	}

	matchers, err := util.CompileIgnorePatterns(ignorePatterns)
	if err != nil {
		return nil, err
	}

	var paths []string
	var walk func(dirSFTP, rel string) error
	walk = func(dirSFTP, rel string) error {
		infos, err := c.sftp.ReadDir(dirSFTP)
		if err != nil {
			return fmt.Errorf("读取远程目录失败: %w", err)
		}
		for _, fi := range infos {
			name := fi.Name()
			if name == "." || name == ".." {
				continue
			}
			entryRel := rel + name
			if fi.IsDir() {
				entryRel += "/"
			}
			matchPath := util.NormalizeIgnoreMatchPath(entryRel)
			if util.ShouldIgnore(name, matchPath, matchers) {
				if onIgnored != nil {
					onIgnored(entryDisplayPath(folderName, rel, name, fi.IsDir(), osType))
				}
				continue
			}
			childSFTP := path.Join(dirSFTP, name)
			if fi.IsDir() {
				if err := walk(childSFTP, rel+name+"/"); err != nil {
					return err
				}
				continue
			}
			paths = append(paths, relPathFromParent(folderName, rel, name, osType))
		}
		return nil
	}
	if err := walk(rootSFTP, ""); err != nil {
		return nil, err
	}
	return paths, nil
}

func relPathFromParent(folderName, internalRel, fileName, osType string) string {
	sep := `/`
	if !model.IsLinuxOS(osType) {
		sep = `\`
	}
	internalRel = strings.ReplaceAll(internalRel, `/`, sep)
	internalRel = strings.TrimSuffix(internalRel, sep)
	parts := []string{folderName}
	if internalRel != "" {
		parts = append(parts, internalRel)
	}
	parts = append(parts, fileName)
	return strings.Join(parts, sep)
}

func entryDisplayPath(folderName, internalRel, name string, isDir bool, osType string) string {
	p := relPathFromParent(folderName, internalRel, name, osType)
	if !isDir {
		return p
	}
	if model.IsLinuxOS(osType) {
		return p + "/"
	}
	return p + `\`
}
