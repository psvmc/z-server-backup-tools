package util

import (
	"path/filepath"
	"strings"
)

func NormalizeRemotePath(p string) string {
	return NormalizeRemotePathForOS(p, "")
}

// NormalizeRemotePathForOS normalizes a remote path for the given OS type.
// Empty osType defaults to Windows (backslash) for backward compatibility.
func NormalizeRemotePathForOS(p string, osType string) string {
	p = strings.TrimSpace(p)
	if strings.EqualFold(strings.TrimSpace(osType), "linux") {
		p = strings.ReplaceAll(p, `\`, `/`)
		if p == "" {
			return ""
		}
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		for strings.Contains(p, "//") {
			p = strings.ReplaceAll(p, "//", "/")
		}
		if len(p) > 1 {
			p = strings.TrimSuffix(p, "/")
		}
		return p
	}
	p = strings.ReplaceAll(p, "/", `\`)
	return p
}

func ToSlashRel(rel string) string {
	return filepath.ToSlash(rel)
}

func JoinRemote(base string, parts ...string) string {
	return JoinRemoteForOS("", base, parts...)
}

func JoinRemoteForOS(osType string, base string, parts ...string) string {
	base = NormalizeRemotePathForOS(base, osType)
	sep := `\`
	if strings.EqualFold(strings.TrimSpace(osType), "linux") {
		sep = `/`
	}
	all := append([]string{base}, parts...)
	return strings.Join(all, sep)
}

func QuoteWindowsArg(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " \t\"") {
		return s
	}
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

// QuoteShellArg quotes s for POSIX shells. Plain tokens are returned as-is;
// otherwise single quotes are used, with internal ' escaped as '\''.
func QuoteShellArg(s string) string {
	if s == "" {
		return `''`
	}
	if !strings.ContainsAny(s, " \t'\"\\$`!*?;&|()<>[]{}#~") {
		return s
	}
	return `'` + strings.ReplaceAll(s, `'`, `'"'"'`) + `'`
}
