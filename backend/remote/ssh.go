package remote

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"z-server-backup-tools/backend/model"
)

type Client struct {
	cfg    model.BackupConfig
	client *ssh.Client
}

func Dial(cfg model.BackupConfig) (*Client, error) {
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	auth := []ssh.AuthMethod{ssh.Password(cfg.Password)}
	hostKeyCallback, err := hostKeyCallback(cfg.KnownHostsFile)
	if err != nil {
		return nil, err
	}
	sshCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            auth,
		HostKeyCallback: hostKeyCallback,
		Timeout:         30 * time.Second,
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return nil, fmt.Errorf("SSH 主机为空，请填写主机地址")
	}
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	client, err := ssh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %w", err)
	}
	return &Client{cfg: cfg, client: client}, nil
}

func (c *Client) Close() error {
	if c.client == nil {
		return nil
	}
	return c.client.Close()
}

func (c *Client) RunRemote(argv ...string) (string, error) {
	out, _, err := c.RunRemoteWithStderr(argv...)
	return out, err
}

func (c *Client) RunRemoteWithTimeout(timeout time.Duration, argv ...string) (string, error) {
	out, _, err := c.RunRemoteWithStderrTimeout(timeout, argv...)
	return out, err
}

func (c *Client) RunRemoteWithStderr(argv ...string) (stdout, stderr string, err error) {
	cmd := BuildRemoteCommand(c.cfg.OSType, c.cfg.RemoteSrv, argv...)
	return c.run(cmd)
}

func (c *Client) RunRemoteWithStderrTimeout(timeout time.Duration, argv ...string) (stdout, stderr string, err error) {
	if timeout <= 0 {
		return c.RunRemoteWithStderr(argv...)
	}
	cmd := BuildRemoteCommand(c.cfg.OSType, c.cfg.RemoteSrv, argv...)
	type result struct {
		out, errOut string
		err         error
	}
	ch := make(chan result, 1)
	go func() {
		out, errOut, runErr := c.run(cmd)
		ch <- result{out: out, errOut: errOut, err: runErr}
	}()
	select {
	case r := <-ch:
		return r.out, r.errOut, r.err
	case <-time.After(timeout):
		_ = c.Close()
		return "", "", fmt.Errorf("SSH 命令超时 (%s)", timeout)
	}
}

func (c *Client) RunShell(cmd string) (string, error) {
	out, _, err := c.run(cmd)
	return out, err
}

func (c *Client) run(cmd string) (stdout, stderr string, err error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", "", err
	}
	defer session.Close()
	var outBuf, errBuf bytes.Buffer
	session.Stdout = &outBuf
	session.Stderr = &errBuf
	if err := session.Run(cmd); err != nil {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", "", fmt.Errorf("%s", msg)
	}
	return strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String()), nil
}

func hostKeyCallback(knownHostsFile string) (ssh.HostKeyCallback, error) {
	_ = knownHostsFile
	return ssh.InsecureIgnoreHostKey(), nil
}
