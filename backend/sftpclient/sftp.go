package sftpclient

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"z-server-backup-tools/backend/model"
)

type Client struct {
	sftp *sftp.Client
}

func Dial(cfg model.BackupConfig) (*Client, error) {
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	sshCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{ssh.Password(cfg.Password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	if strings.TrimSpace(cfg.Host) == "" {
		return nil, fmt.Errorf("SSH 主机为空，请填写主机地址")
	}
	conn, err := ssh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %w", err)
	}
	sc, err := sftp.NewClient(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return &Client{sftp: sc}, nil
}

func (c *Client) Close() error {
	return c.sftp.Close()
}

func (c *Client) Download(ctx context.Context, remotePath, localPath string) error {
	return c.DownloadWithProgress(ctx, remotePath, localPath, 0, nil)
}

func (c *Client) DownloadWithProgress(ctx context.Context, remotePath, localPath string, total int64, onProgress ProgressFunc) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}
	tmp := localPath + ".downloading"
	rf, err := c.sftp.Open(remotePath)
	if err != nil {
		return err
	}
	defer rf.Close()
	if total <= 0 {
		if fi, statErr := rf.Stat(); statErr == nil {
			total = fi.Size()
		}
	}
	lf, err := os.Create(tmp)
	if err != nil {
		return err
	}

	pw := &progressWriter{w: lf, total: total, onProg: onProgress}
	done := make(chan error, 1)
	go func() {
		_, err := io.Copy(pw, rf)
		done <- err
	}()
	select {
	case <-ctx.Done():
		lf.Close()
		os.Remove(tmp)
		return ctx.Err()
	case err := <-done:
		if err != nil {
			lf.Close()
			os.Remove(tmp)
			return err
		}
	}
	if err := lf.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Remove(localPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmp, localPath)
}

func (c *Client) Remove(remotePath string) error {
	return c.sftp.Remove(remotePath)
}
