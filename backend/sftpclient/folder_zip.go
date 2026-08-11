package sftpclient

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"z-server-backup-tools/backend/model"
	"z-server-backup-tools/backend/util"
)

type zipEntry struct {
	remoteSFTP string
	name       string
	size       int64
	isDir      bool
}

// ZipRemoteFolder downloads a remote directory via SFTP and writes a local zip file.
// Zip entries include the folder name as the top-level directory (e.g. myapp/foo.txt).
// ignorePatterns are regexes matched against each entry basename or relative path.
func (c *Client) ZipRemoteFolder(ctx context.Context, remoteDir, osType, localZipPath string, ignorePatterns []string, onIgnored func(string), onProgress ProgressFunc) error {
	osType = model.NormalizeOSType(osType)
	remoteDir = strings.TrimSpace(remoteDir)
	if remoteDir == "" {
		return fmt.Errorf("远程文件夹路径不能为空")
	}
	folderName, err := remoteFolderBaseName(remoteDir)
	if err != nil {
		return err
	}
	rootSFTP := toSFTPPath(remoteDir, osType)
	info, err := c.sftp.Stat(rootSFTP)
	if err != nil {
		return fmt.Errorf("读取远程文件夹失败: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("远程路径不是文件夹: %s", remoteDir)
	}

	entries, totalBytes, err := c.collectZipEntries(rootSFTP, folderName, osType, ignorePatterns, onIgnored)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		entries = append(entries, zipEntry{name: folderName + "/", isDir: true})
	}

	if err := os.MkdirAll(filepath.Dir(localZipPath), 0o755); err != nil {
		return err
	}
	tmp := localZipPath + ".downloading"
	zf, err := os.Create(tmp)
	if err != nil {
		return err
	}

	zw := zip.NewWriter(zf)
	var written int64

	for _, ent := range entries {
		select {
		case <-ctx.Done():
			zw.Close()
			zf.Close()
			os.Remove(tmp)
			return ctx.Err()
		default:
		}
		if ent.isDir {
			if _, err := zw.Create(ent.name); err != nil {
				zw.Close()
				zf.Close()
				os.Remove(tmp)
				return err
			}
			continue
		}
		hdr := &zip.FileHeader{
			Name:   ent.name,
			Method: zip.Deflate,
		}
		hdr.SetModTime(time.Now())
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			zw.Close()
			zf.Close()
			os.Remove(tmp)
			return err
		}
		rf, err := c.sftp.Open(ent.remoteSFTP)
		if err != nil {
			zw.Close()
			zf.Close()
			os.Remove(tmp)
			return fmt.Errorf("打开远程文件失败 %s: %w", ent.remoteSFTP, err)
		}
		pw := &progressWriter{
			w:     w,
			total: ent.size,
			onProg: func(done, _ int64) {
				if onProgress != nil {
					onProgress(written+done, totalBytes)
				}
			},
		}
		n, copyErr := io.Copy(pw, rf)
		rf.Close()
		if copyErr != nil {
			zw.Close()
			zf.Close()
			os.Remove(tmp)
			return copyErr
		}
		written += n
		if onProgress != nil {
			onProgress(written, totalBytes)
		}
	}

	if err := zw.Close(); err != nil {
		zf.Close()
		os.Remove(tmp)
		return err
	}
	if err := zf.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Remove(localZipPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmp, localZipPath)
}

func remoteFolderBaseName(remoteDir string) (string, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(remoteDir), `\`, `/`)
	normalized = strings.TrimSuffix(normalized, `/`)
	if normalized == "" {
		return "", fmt.Errorf("远程文件夹路径不能为空")
	}
	base := filepath.Base(filepath.FromSlash(normalized))
	if base == "." || base == string(filepath.Separator) || base == "/" || base == `\` {
		return "", fmt.Errorf("无法从远程路径解析文件夹名")
	}
	return base, nil
}

func (c *Client) collectZipEntries(rootSFTP, folderName, osType string, ignorePatterns []string, onIgnored func(string)) ([]zipEntry, int64, error) {
	matchers, err := util.CompileIgnorePatterns(ignorePatterns)
	if err != nil {
		return nil, 0, err
	}
	var out []zipEntry
	var total int64
	var walk func(dirSFTP, rel string) error
	walk = func(dirSFTP, rel string) error {
		infos, err := c.sftp.ReadDir(dirSFTP)
		if err != nil {
			return fmt.Errorf("读取远程目录失败: %w", err)
		}
		if len(infos) == 0 && rel != "" {
			out = append(out, zipEntry{name: folderName + "/" + rel, isDir: true})
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
				childRel := rel + name + "/"
				if err := walk(childSFTP, childRel); err != nil {
					return err
				}
				continue
			}
			zipName := folderName + "/"
			if rel != "" {
				zipName += rel
			}
			zipName += name
			size := fi.Size()
			out = append(out, zipEntry{remoteSFTP: childSFTP, name: zipName, size: size})
			total += size
		}
		return nil
	}
	if err := walk(rootSFTP, ""); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}
