package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"z-server-backup-tools/backend/zipbak"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "init":
		runInit(os.Args[2:])
	case "pack":
		runPack(os.Args[2:])
	case "pack-ahead":
		runPackAhead(os.Args[2:])
	case "ack":
		runAck(os.Args[2:])
	case "status":
		runStatus(os.Args[2:])
	case "reset":
		runReset(os.Args[2:])
	case "oversized":
		runOversized(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "用法: zipbak-srv init|pack|pack-ahead|ack|status|reset|oversized [flags]\n")
}

func runInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	dir := fs.String("dir", "", "源目录")
	state := fs.String("state", "", "state.db 路径（SQLite）")
	staging := fs.String("staging", "", "staging 目录")
	prefix := fs.String("prefix", "", "分卷文件名前缀")
	_ = fs.Parse(args)
	if _, err := zipbak.InitState(*dir, *state, *staging, *prefix); err != nil {
		fail(err)
	}
}

func runPack(args []string) {
	fs := flag.NewFlagSet("pack", flag.ExitOnError)
	state := fs.String("state", "", "state.db 路径（SQLite）")
	maxGB := fs.Float64("max-gb", 2, "每卷上限 GB")
	_ = fs.Parse(args)
	maxBytes := zipbak.MaxPartBytesFromGB(*maxGB)
	path, err := zipbak.Pack(*state, maxBytes)
	if err != nil {
		fail(err)
	}
	fmt.Println(path)
}

func runPackAhead(args []string) {
	fs := flag.NewFlagSet("pack-ahead", flag.ExitOnError)
	state := fs.String("state", "", "state.db 路径（SQLite）")
	maxGB := fs.Float64("max-gb", 2, "每卷上限 GB")
	_ = fs.Parse(args)
	maxBytes := zipbak.MaxPartBytesFromGB(*maxGB)
	path, err := zipbak.PackAhead(*state, maxBytes)
	if err != nil {
		fail(err)
	}
	if path != "" {
		fmt.Println(path)
	}
}

func runAck(args []string) {
	fs := flag.NewFlagSet("ack", flag.ExitOnError)
	state := fs.String("state", "", "state.db 路径（SQLite）")
	_ = fs.Parse(args)
	if err := zipbak.Ack(*state); err != nil {
		fail(err)
	}
}

func runStatus(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	state := fs.String("state", "", "state.db 路径（SQLite）")
	maxGB := fs.Float64("max-gb", 2, "每卷上限 GB")
	_ = fs.Parse(args)
	maxBytes := zipbak.MaxPartBytesFromGB(*maxGB)
	st, err := zipbak.ReadStatus(*state, maxBytes)
	if err != nil {
		fail(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(st)
}

func runReset(args []string) {
	fs := flag.NewFlagSet("reset", flag.ExitOnError)
	state := fs.String("state", "", "state.db 路径（SQLite）")
	_ = fs.Parse(args)
	if err := zipbak.ResetProgress(*state); err != nil {
		fail(err)
	}
}

func runOversized(args []string) {
	fs := flag.NewFlagSet("oversized", flag.ExitOnError)
	state := fs.String("state", "", "state.db 路径（SQLite）")
	maxGB := fs.Float64("max-gb", 2, "每卷上限 GB")
	_ = fs.Parse(args)
	maxBytes := zipbak.MaxPartBytesFromGB(*maxGB)
	items, err := zipbak.ListOversizedFiles(*state, maxBytes)
	if err != nil {
		fail(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(items)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
