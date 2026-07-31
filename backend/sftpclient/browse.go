package sftpclient

import (
	"fmt"
	"path"
	"strings"
	"unicode"
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
	// /C: or /C:/ → drive root, parent is drive list
	trimmed := strings.TrimSuffix(sftpPath, "/")
	if isSFTPDriveRoot(trimmed) {
		return "/"
	}
	parent := path.Dir(sftpPath)
	if parent == "." {
		return "/"
	}
	return parent
}

func isSFTPDriveRoot(sftpPath string) bool {
	p := strings.TrimSuffix(strings.TrimSpace(sftpPath), "/")
	if strings.HasPrefix(p, "/") {
		p = p[1:]
	}
	return isDriveName(p)
}

func isDriveName(name string) bool {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, `\`)
	name = strings.TrimSuffix(name, `/`)
	if len(name) == 2 && name[1] == ':' && unicode.IsLetter(rune(name[0])) {
		return true
	}
	return false
}

func (c *Client) ListDirectories(windowsPath string) (current string, parent string, names []string, err error) {
	hint := strings.TrimSpace(windowsPath)
	if hint == "" || hint == "/" || hint == `\` {
		return c.listDriveRoots()
	}

	sftpPath := toSFTPPath(hint)
	infos, err := c.sftp.ReadDir(sftpPath)
	if err != nil {
		return "", "", nil, fmt.Errorf("读取远程目录失败: %w", err)
	}
	current = fromSFTPPath(sftpPath)
	parentSFTP := parentSFTPPath(sftpPath)
	if parentSFTP == "/" {
		parent = ""
	} else {
		parent = fromSFTPPath(parentSFTP)
	}
	out := make([]string, 0, len(infos))
	for _, fi := range infos {
		if fi.IsDir() && fi.Name() != "." && fi.Name() != ".." {
			out = append(out, fi.Name())
		}
	}
	return current, parent, out, nil
}

func (c *Client) listDriveRoots() (current string, parent string, names []string, err error) {
	seen := map[string]struct{}{}
	var out []string

	add := func(name string) {
		name = strings.TrimSpace(name)
		name = strings.TrimSuffix(name, `/`)
		name = strings.TrimSuffix(name, `\`)
		if !isDriveName(name) {
			// OpenSSH 有时返回 "C" 而不是 "C:"
			if len(name) == 1 && unicode.IsLetter(rune(name[0])) {
				name = strings.ToUpper(name) + ":"
			} else {
				return
			}
		} else {
			name = strings.ToUpper(name[:1]) + ":"
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}

	if infos, readErr := c.sftp.ReadDir("/"); readErr == nil {
		for _, fi := range infos {
			if fi.Name() == "." || fi.Name() == ".." {
				continue
			}
			add(fi.Name())
		}
	}

	// 回退：探测常见盘符是否存在
	if len(out) == 0 {
		for letter := 'A'; letter <= 'Z'; letter++ {
			drive := string(letter) + ":"
			candidates := []string{"/" + drive + "/", "/" + drive, "/" + string(letter)}
			for _, p := range candidates {
				if _, statErr := c.sftp.Stat(p); statErr == nil {
					add(drive)
					break
				}
			}
		}
	}

	if len(out) == 0 {
		return "", "", nil, fmt.Errorf("无法列出远程盘符，请手动输入路径（如 D:\\Tools\\zipbak）")
	}
	return "", "", out, nil
}

func (c *Client) JoinRemoteDir(baseWindows, name string) string {
	return JoinRemoteDirPublic(baseWindows, name)
}

func joinRemotePath(baseWindows, name string) string {
	base := strings.TrimSpace(baseWindows)
	name = strings.Trim(name, `\/`)
	if base == "" {
		return strings.ReplaceAll(name, `/`, `\`)
	}
	base = strings.TrimSuffix(base, `\`)
	base = strings.TrimSuffix(base, `/`)
	return base + `\` + name
}

// JoinRemoteDirPublic joins using Windows-style separators for display.
func JoinRemoteDirPublic(baseWindows, name string) string {
	return joinRemotePath(baseWindows, name)
}
