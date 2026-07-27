package sftpclient

import (
	"fmt"
	"path"
	"strings"
)

func toSFTPPath(windowsPath string) string {
	p := strings.TrimSpace(windowsPath)
	p = strings.ReplaceAll(p, `\`, `/`)
	if p == "" || p == "/" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		if len(p) >= 2 && p[1] == ':' {
			return "/" + p
		}
		return "/" + strings.TrimPrefix(p, "/")
	}
	return p
}

func fromSFTPPath(sftpPath string) string {
	p := strings.TrimSpace(sftpPath)
	if p == "" || p == "/" {
		return ""
	}
	p = strings.TrimPrefix(p, "/")
	return strings.ReplaceAll(p, `/`, `\`)
}

func parentSFTPPath(sftpPath string) string {
	if sftpPath == "" || sftpPath == "/" {
		return ""
	}
	parent := path.Dir(sftpPath)
	if parent == "." {
		return "/"
	}
	return parent
}

func (c *Client) ListDirectories(windowsPath string) (current string, parent string, names []string, err error) {
	sftpPath := toSFTPPath(windowsPath)
	infos, err := c.sftp.ReadDir(sftpPath)
	if err != nil {
		return "", "", nil, fmt.Errorf("读取远程目录失败: %w", err)
	}
	current = fromSFTPPath(sftpPath)
	parent = fromSFTPPath(parentSFTPPath(sftpPath))
	out := make([]string, 0, len(infos))
	for _, fi := range infos {
		if fi.IsDir() && fi.Name() != "." && fi.Name() != ".." {
			out = append(out, fi.Name())
		}
	}
	return current, parent, out, nil
}

func (c *Client) JoinRemoteDir(baseWindows, name string) string {
	return JoinRemoteDirPublic(baseWindows, name)
}

func joinRemotePath(baseWindows, name string) string {
	base := strings.TrimSpace(baseWindows)
	if base == "" {
		return strings.ReplaceAll(name, `/`, `\`)
	}
	base = strings.TrimSuffix(base, `\`)
	base = strings.TrimSuffix(base, `/`)
	return base + `\` + strings.Trim(name, `\/`)
}

// JoinRemoteDirPublic joins using Windows-style separators for display.
func JoinRemoteDirPublic(baseWindows, name string) string {
	return joinRemotePath(baseWindows, name)
}
