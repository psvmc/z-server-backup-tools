package model

import "testing"

func TestServerApplyTo(t *testing.T) {
	base := BackupConfig{NotifyEmail: "a@b.c", SmtpHost: "smtp", RemoteSource: "keep"}
	srv := Server{
		Host: "10.0.0.1", Port: 22, User: "u", Password: "p",
		SupportMultiFile: true, RemoteAppDir: `D:\zipbak`, MaxPartGB: 3,
	}
	out := srv.ApplyTo(base)
	if out.Host != "10.0.0.1" || out.RemoteAppDir != `D:\zipbak` || out.MaxPartGB != 3 {
		t.Fatalf("ssh/app: %+v", out)
	}
	if out.NotifyEmail != "a@b.c" || out.RemoteSource != "keep" {
		t.Fatalf("should keep notify/task fields: %+v", out)
	}
}
