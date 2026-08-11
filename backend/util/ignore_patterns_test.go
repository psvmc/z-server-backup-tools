package util

import (
	"testing"
)

func TestCompileIgnorePatterns(t *testing.T) {
	matchers, err := CompileIgnorePatterns([]string{`\.log$`, `^tmp$`, "", `\.log$`})
	if err != nil {
		t.Fatal(err)
	}
	if len(matchers) != 2 {
		t.Fatalf("got %d matchers", len(matchers))
	}
	if !ShouldIgnore("app.log", "logs/app.log", matchers) {
		t.Fatal("expected basename match")
	}
	if !ShouldIgnore("tmp", "cache/tmp", matchers) {
		t.Fatal("expected basename match for tmp")
	}
	if ShouldIgnore("app.txt", "logs/app.txt", matchers) {
		t.Fatal("expected no match")
	}
}

func TestCompileIgnorePatternsGlob(t *testing.T) {
	matchers, err := CompileIgnorePatterns([]string{`*log.txt*`})
	if err != nil {
		t.Fatal(err)
	}
	if !ShouldIgnore("log.txt", "log.txt", matchers) {
		t.Fatal("expected exact log.txt match")
	}
	if !ShouldIgnore("Log.TXT", "Log.TXT", matchers) {
		t.Fatal("expected case-insensitive match")
	}
	if !ShouldIgnore("app.log.txt.bak", "logs/app.log.txt.bak", matchers) {
		t.Fatal("expected glob match")
	}
	if ShouldIgnore("app.log", "logs/app.log", matchers) {
		t.Fatal("expected no match for app.log")
	}
}

func TestCompileIgnorePatternsEmpty(t *testing.T) {
	m, err := CompileIgnorePatterns(nil)
	if err != nil || m != nil {
		t.Fatalf("got %v %v", m, err)
	}
}
