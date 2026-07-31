package sftpclient

import (
	"fmt"
	"path"
	"strings"
	"unicode"

	"z-server-backup-tools/backend/model"
)

// RemoteListEntry is one directory or file returned by ListEntries.
type RemoteListEntry struct {
	Name  string
	IsDir bool
}

func toSFTPPath(remotePath string, osType string) string {
	p := strings.TrimSpace(remotePath)
	if model.IsLinuxOS(osType) {
		p = strings.ReplaceAll(p, `\`, `/`)
		if p == "" || p == "/" {
			return "/"
		}
		if !strings.HasPrefix(p, "/") {
			return "/" + strings.TrimPrefix(p, "/")
		}
		return p
	}
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

func fromSFTPPath(sftpPath string, osType string) string {
	p := strings.TrimSpace(sftpPath)
	if model.IsLinuxOS(osType) {
		if p == "" || p == "/" {
			return "/"
		}
		return p
	}
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

// ListEntries lists remote directories (and files) at pathHint.
// Linux: empty or "/" lists filesystem root "/".
// Windows: empty lists drive letters.
func (c *Client) ListEntries(pathHint string, osType string) (current string, parent string, entries []RemoteListEntry, err error) {
	osType = model.NormalizeOSType(osType)
	hint := strings.TrimSpace(pathHint)

	if model.IsLinuxOS(osType) {
		if hint == "" {
			hint = "/"
		}
		return c.listLinuxEntries(hint)
	}

	if hint == "" || hint == "/" || hint == `\` {
		return c.listDriveRootEntries()
	}
	return c.listWindowsEntries(hint)
}

func (c *Client) listLinuxEntries(hint string) (current string, parent string, entries []RemoteListEntry, err error) {
	sftpPath := toSFTPPath(hint, model.OSTypeLinux)
	infos, err := c.sftp.ReadDir(sftpPath)
	if err != nil {
		return "", "", nil, fmt.Errorf("读取远程目录失败: %w", err)
	}
	current = fromSFTPPath(sftpPath, model.OSTypeLinux)
	if current == "/" {
		parent = ""
	} else {
		parent = path.Dir(current)
		if parent == "." {
			parent = "/"
		}
	}
	out := make([]RemoteListEntry, 0, len(infos))
	for _, fi := range infos {
		name := fi.Name()
		if name == "." || name == ".." {
			continue
		}
		out = append(out, RemoteListEntry{Name: name, IsDir: fi.IsDir()})
	}
	return current, parent, out, nil
}

func (c *Client) listWindowsEntries(hint string) (current string, parent string, entries []RemoteListEntry, err error) {
	sftpPath := toSFTPPath(hint, model.OSTypeWindows)
	infos, err := c.sftp.ReadDir(sftpPath)
	if err != nil {
		return "", "", nil, fmt.Errorf("读取远程目录失败: %w", err)
	}
	current = fromSFTPPath(sftpPath, model.OSTypeWindows)
	parentSFTP := parentSFTPPath(sftpPath)
	if parentSFTP == "/" {
		parent = ""
	} else {
		parent = fromSFTPPath(parentSFTP, model.OSTypeWindows)
	}
	out := make([]RemoteListEntry, 0, len(infos))
	for _, fi := range infos {
		name := fi.Name()
		if name == "." || name == ".." {
			continue
		}
		out = append(out, RemoteListEntry{Name: name, IsDir: fi.IsDir()})
	}
	return current, parent, out, nil
}

func (c *Client) listDriveRootEntries() (current string, parent string, entries []RemoteListEntry, err error) {
	_, _, names, err := c.listDriveRoots()
	if err != nil {
		return "", "", nil, err
	}
	out := make([]RemoteListEntry, 0, len(names))
	for _, name := range names {
		out = append(out, RemoteListEntry{Name: name, IsDir: true})
	}
	return "", "", out, nil
}

// ListDirectories lists only directories (Windows-compatible helper).
func (c *Client) ListDirectories(windowsPath string) (current string, parent string, names []string, err error) {
	current, parent, entries, err := c.ListEntries(windowsPath, model.OSTypeWindows)
	if err != nil {
		return "", "", nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir {
			out = append(out, e.Name)
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
	return JoinRemotePath(baseWindows, name, model.OSTypeWindows)
}

func joinRemotePathWindows(baseWindows, name string) string {
	base := strings.TrimSpace(baseWindows)
	name = strings.Trim(name, `\/`)
	if base == "" {
		return strings.ReplaceAll(name, `/`, `\`)
	}
	base = strings.TrimSuffix(base, `\`)
	base = strings.TrimSuffix(base, `/`)
	return base + `\` + name
}

// JoinRemotePath joins a remote path segment using separators for osType.
func JoinRemotePath(base, name, osType string) string {
	if model.IsLinuxOS(osType) {
		base = strings.TrimSpace(base)
		name = strings.Trim(name, `/`)
		if name == "" {
			if base == "" {
				return "/"
			}
			return base
		}
		if base == "" || base == "/" {
			return "/" + name
		}
		base = strings.TrimSuffix(base, "/")
		return base + "/" + name
	}
	return joinRemotePathWindows(base, name)
}

// JoinRemoteDirPublic joins using Windows-style separators for display.
func JoinRemoteDirPublic(baseWindows, name string) string {
	return JoinRemotePath(baseWindows, name, model.OSTypeWindows)
}
