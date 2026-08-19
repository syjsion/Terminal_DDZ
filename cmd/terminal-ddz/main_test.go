package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "terminal-ddz") {
		t.Fatalf("version output = %q", stdout.String())
	}
}

func TestUnknownArgument(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"unexpected"}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "未知参数") {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr.String())
	}
}

func TestHelpFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--help"}, &stdout, &stderr); code != 0 || !strings.Contains(stderr.String(), "用法") {
		t.Fatalf("exit code = %d, stderr=%s", code, stderr.String())
	}
}
