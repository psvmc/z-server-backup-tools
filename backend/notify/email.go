package notify

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"z-server-backup-tools/backend/model"
)

type BackupResult struct {
	Success      bool
	Host         string
	RemoteSource string
	LocalDir     string
	TotalFiles   int
	PackedFiles  int
	Error        string
}

// SendBackupNotification emails the recipient when SMTP is configured.
func SendBackupNotification(cfg model.BackupConfig, result BackupResult) error {
	to := strings.TrimSpace(cfg.NotifyEmail)
	if to == "" {
		return nil
	}
	subject, body := buildMessage(result)
	return sendConfiguredMail(cfg, to, subject, body)
}

// SendSingleFileNotification emails when a single-file download finishes or fails.
func SendSingleFileNotification(cfg model.BackupConfig, success bool, remoteFile, localPath, errMsg string) error {
	to := strings.TrimSpace(cfg.NotifyEmail)
	if to == "" {
		return nil
	}
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		host = "（未知）"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	var subject, body string
	if success {
		subject = fmt.Sprintf("[单文件下载完成] %s", host)
		body = fmt.Sprintf("单文件下载已完成。\n\n时间：%s\n主机：%s\n远程文件：%s\n本机文件：%s\n", now, host, remoteFile, localPath)
	} else {
		subject = fmt.Sprintf("[单文件下载异常] %s", host)
		body = fmt.Sprintf("单文件下载异常停止。\n\n时间：%s\n主机：%s\n远程文件：%s\n本机文件：%s\n错误：%s\n", now, host, remoteFile, localPath, errMsg)
	}
	return sendConfiguredMail(cfg, to, subject, body)
}

// SendTestEmail sends a simple test message using the configured SMTP settings.
func SendTestEmail(cfg model.BackupConfig) error {
	to := strings.TrimSpace(cfg.NotifyEmail)
	if to == "" {
		return fmt.Errorf("通知邮箱不能为空")
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	subject := "[邮箱测试] 服务器文件备份"
	body := fmt.Sprintf("这是一封测试邮件，说明 SMTP 配置可用。\n\n发送时间：%s\n", now)
	return sendConfiguredMail(cfg, to, subject, body)
}

func sendConfiguredMail(cfg model.BackupConfig, to, subject, body string) error {
	host := strings.TrimSpace(cfg.SmtpHost)
	if host == "" {
		return fmt.Errorf("未配置 SMTP 服务器，无法发送邮件")
	}
	port := cfg.SmtpPort
	if port <= 0 {
		port = 465
	}
	user := strings.TrimSpace(cfg.SmtpUser)
	pass := cfg.SmtpPassword
	from := user
	if from == "" {
		from = to
	}
	msg := buildMIME(from, to, subject, body)
	addr := fmt.Sprintf("%s:%d", host, port)
	return sendMail(addr, host, port, user, pass, from, []string{to}, msg)
}

func buildMessage(r BackupResult) (subject, body string) {
	host := r.Host
	if host == "" {
		host = "（未知）"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	if r.Success {
		subject = fmt.Sprintf("[备份完成] %s", host)
		body = fmt.Sprintf(
			"备份任务已成功完成。\n\n时间：%s\n主机：%s\n远程源目录：%s\n本机保存目录：%s\n文件进度：%d / %d\n",
			now, host, r.RemoteSource, r.LocalDir, r.PackedFiles, r.TotalFiles,
		)
		return subject, body
	}
	subject = fmt.Sprintf("[备份异常] %s", host)
	errText := strings.TrimSpace(r.Error)
	if errText == "" {
		errText = "未知错误"
	}
	body = fmt.Sprintf(
		"备份任务异常停止。\n\n时间：%s\n主机：%s\n远程源目录：%s\n本机保存目录：%s\n文件进度：%d / %d\n错误：%s\n",
		now, host, r.RemoteSource, r.LocalDir, r.PackedFiles, r.TotalFiles, errText,
	)
	return subject, body
}

func encodeSubject(s string) string {
	if s == "" {
		return s
	}
	for _, r := range s {
		if r > 127 {
			return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(s)) + "?="
		}
	}
	return s
}

func buildMIME(from, to, subject, body string) []byte {
	headers := []string{
		"From: " + from,
		"To: " + to,
		"Subject: " + encodeSubject(subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}
	return []byte(strings.Join(headers, "\r\n"))
}

func sendMail(addr, host string, port int, user, pass, from string, to []string, msg []byte) error {
	var (
		conn net.Conn
		err  error
	)
	// 465：隐式 SSL；其余端口先明文再视情况 STARTTLS
	if port == 465 {
		conn, err = tls.DialWithDialer(&net.Dialer{Timeout: 30 * time.Second}, "tcp", addr, &tls.Config{ServerName: host})
	} else {
		conn, err = net.DialTimeout("tcp", addr, 30*time.Second)
	}
	if err != nil {
		return fmt.Errorf("连接 SMTP 失败: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP 握手失败: %w", err)
	}
	defer c.Close()

	if port != 465 {
		if ok, _ := c.Extension("STARTTLS"); ok {
			if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
				return fmt.Errorf("STARTTLS 失败: %w", err)
			}
		}
	}

	if user != "" {
		if ok, _ := c.Extension("AUTH"); !ok {
			return fmt.Errorf("服务器不支持 AUTH，请检查 SMTP 配置")
		}
		auth := smtp.PlainAuth("", user, pass, host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("SMTP 登录失败: %w", err)
		}
	}

	if err := c.Mail(from); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("设置收件人失败: %w", err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("开始发送正文失败: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("写入邮件失败: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("结束正文失败: %w", err)
	}
	if err := c.Quit(); err != nil {
		// 部分服务器在 Quit 时报错但邮件已发出，忽略
		return nil
	}
	return nil
}
