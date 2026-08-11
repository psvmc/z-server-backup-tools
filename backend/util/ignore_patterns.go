package util

import (
	"fmt"
	"regexp"
	"strings"
)

// NormalizeIgnorePatterns trims, deduplicates, and drops empty ignore pattern strings.
func NormalizeIgnorePatterns(patterns []string) []string {
	seen := make(map[string]struct{}, len(patterns))
	out := make([]string, 0, len(patterns))
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// globToRegex converts shell-style wildcards to a partial-match regex.
// * -> .*, ? -> ., other regex metacharacters are escaped.
func globToRegex(glob string) string {
	var b strings.Builder
	for _, r := range glob {
		switch r {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteByte('.')
		case '.', '+', '(', ')', '|', '^', '$', '[', ']', '{', '}', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func compileIgnorePattern(p string) (*regexp.Regexp, error) {
	var src string
	switch {
	case strings.ContainsAny(p, "*?"):
		// *log.txt* 等通配符：优先按 glob 解析，避免误走正则
		src = globToRegex(p)
	default:
		if _, err := regexp.Compile(p); err == nil {
			src = p
		} else {
			src = globToRegex(p)
		}
	}
	// 文件名忽略：大小写不敏感（Windows 常见 Log.txt / log.txt）
	re, err := regexp.Compile("(?i)" + src)
	if err != nil {
		return nil, fmt.Errorf("忽略规则无效 %q（支持正则或通配符 * ?）", p)
	}
	return re, nil
}

// CompileIgnorePatterns compiles ignore patterns (glob * / ? or Go regex).
func CompileIgnorePatterns(patterns []string) ([]*regexp.Regexp, error) {
	normalized := NormalizeIgnorePatterns(patterns)
	if len(normalized) == 0 {
		return nil, nil
	}
	out := make([]*regexp.Regexp, 0, len(normalized))
	for _, p := range normalized {
		re, err := compileIgnorePattern(p)
		if err != nil {
			return nil, err
		}
		out = append(out, re)
	}
	return out, nil
}

// ShouldIgnore reports whether a file or directory should be skipped.
// basename is the entry name; relPath is the path relative to the backup folder root (e.g. logs/app.log).
func ShouldIgnore(basename, relPath string, matchers []*regexp.Regexp) bool {
	if len(matchers) == 0 {
		return false
	}
	basename = strings.TrimSpace(basename)
		relPath = NormalizeIgnoreMatchPath(relPath)
	for _, re := range matchers {
		if basename != "" && re.MatchString(basename) {
			return true
		}
		if relPath != "" && re.MatchString(relPath) {
			return true
		}
	}
	return false
}

// NormalizeIgnoreMatchPath normalizes a relative path for ignore matching.
func NormalizeIgnoreMatchPath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.TrimSuffix(p, `/`)
	p = strings.TrimSuffix(p, `\`)
	return strings.ReplaceAll(p, `\`, `/`)
}
