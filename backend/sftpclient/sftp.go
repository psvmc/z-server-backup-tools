package sftpclient

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"z-server-backup-tools/backend/model"
	"z-server-backup-tools/backend/sshdial"
)

type Client struct {
	ssh  *ssh.Client // Dial 创建时持有，Close 时一并关闭；NewFromConn 为 nil
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
	if strings.TrimSpace(cfg.Host) == "" {
		return nil, fmt.Errorf("SSH 主机为空，请填写主机地址")
	}
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	release := sshdial.Acquire(addr)
	conn, err := ssh.Dial("tcp", addr, sshCfg)
	release()
	if err != nil {
		sshdial.NoteFailure(addr)
		return nil, fmt.Errorf("SSH 连接失败: %w", err)
	}
	sc, err := sftp.NewClient(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &Client{ssh: conn, sftp: sc}, nil
}

// NewFromConn 基于已有 SSH 连接创建 SFTP 客户端（不关闭底层 conn）。
func NewFromConn(conn *ssh.Client) (*Client, error) {
	sc, err := sftp.NewClient(conn)
	if err != nil {
		return nil, err
	}
	return &Client{sftp: sc}, nil
}

func (c *Client) Close() error {
	var first error
	if c.sftp != nil {
		if err := c.sftp.Close(); err != nil && first == nil {
			first = err
		}
		c.sftp = nil
	}
	if c.ssh != nil {
		if err := c.ssh.Close(); err != nil && first == nil {
			first = err
		}
		c.ssh = nil
	}
	return first
}

// RemoveOnConn 在已有 SSH 连接上删除远程文件，不关闭底层连接。
func RemoveOnConn(conn *ssh.Client, remotePath string) error {
	sc, err := sftp.NewClient(conn)
	if err != nil {
		return err
	}
	defer sc.Close()
	return sc.Remove(remotePath)
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

// WriteRemoteFile creates or overwrites a remote file with the given content.
func (c *Client) WriteRemoteFile(remotePath string, data []byte) error {
	f, err := c.sftp.Create(remotePath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}
