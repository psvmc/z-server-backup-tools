package util

import (
	"path/filepath"
	"strings"
)

func NormalizeRemotePath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, "/", `\`)
	return p
}

func ToSlashRel(rel string) string {
	return filepath.ToSlash(rel)
}

func JoinRemote(base string, parts ...string) string {
	all := append([]string{NormalizeRemotePath(base)}, parts...)
	return filepath.Join(all...)
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
