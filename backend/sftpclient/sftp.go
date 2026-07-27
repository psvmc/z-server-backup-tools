package sftpclient

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}
	tmp := localPath + ".downloading"
	rf, err := c.sftp.Open(remotePath)
	if err != nil {
		return err
	}
	defer rf.Close()
	lf, err := os.Create(tmp)
	if err != nil {
		return err
	}

	// 使用 io.Copy 配合 context 取消检测，通过 goroutine + select 实现
	done := make(chan error, 1)
	go func() {
		_, err := io.Copy(lf, rf)
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
