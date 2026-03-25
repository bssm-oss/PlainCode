package main

import (
	"os"
	"strings"
	"testing"
)

func TestNormalizeLanguage(t *testing.T) {
	cases := map[string]string{
		"":            "",
		"ko":          "ko",
		"ko_KR.UTF-8": "ko",
		"en":          "en",
		"en_US.UTF-8": "en",
		"fr":          "",
	}

	for input, want := range cases {
		if got := normalizeLanguage(input); got != want {
			t.Fatalf("normalizeLanguage(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestResolveHelpLanguageFromArgs(t *testing.T) {
	t.Setenv("LANG", "en_US.UTF-8")

	if got := resolveHelpLanguage([]string{"--lang", "ko"}); got != "ko" {
		t.Fatalf("resolveHelpLanguage(--lang ko) = %q, want ko", got)
	}
	if got := resolveHelpLanguage([]string{"-l", "en"}); got != "en" {
		t.Fatalf("resolveHelpLanguage(-l en) = %q, want en", got)
	}
}

func TestResolveHelpLanguageFromEnvironment(t *testing.T) {
	t.Setenv("PLAINCODE_LANG", "")
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_MESSAGES", "")
	t.Setenv("LANG", "ko_KR.UTF-8")

	if got := resolveHelpLanguage(nil); got != "ko" {
		t.Fatalf("resolveHelpLanguage() = %q, want ko", got)
	}
}

func TestUsageTextSupportsLanguages(t *testing.T) {
	ko := usageText("ko")
	if !strings.Contains(ko, "사용법: plaincode <command> [options]") {
		t.Fatalf("korean usage missing expected text:\n%s", ko)
	}

	en := usageText("en")
	if !strings.Contains(en, "Usage: plaincode <command> [options]") {
		t.Fatalf("english usage missing expected text:\n%s", en)
	}
}

func TestMainHelpAliases(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	for _, args := range [][]string{
		{"plaincode", "help", "--lang", "ko"},
		{"plaincode", "--help", "--lang", "en"},
		{"plaincode", "-h", "--lang", "ko"},
	} {
		os.Args = args
		main()
	}
}
