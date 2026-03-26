package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func printUsage(args []string) {
	fmt.Print(usageText(resolveHelpLanguage(args)))
}

func usageText(lang string) string {
	switch lang {
	case "ko":
		return `plaincode — 스펙 기반 멀티 에이전트 빌드 도구

사용법: plaincode <command> [options]
도움말: plaincode help [--lang ko|en]

핵심 명령:
  init                        새 PlainCode 프로젝트 초기화
  build [--spec <id>]         spec를 코드로 빌드
  test --spec <id>            spec 기준으로 구현 검증
  run --spec <id>             빌드된 서비스를 실행
  stop --spec <id>            실행 중인 서비스를 중지
  status [--spec <id>]        서비스 실행 상태 확인
  logs --spec <id>            저장된 실행 로그 / 이벤트 확인
  change -m "설명"            spec가 아니라 구현을 수정해야 할 때 사용
  takeover <file|package>     기존 코드에서 spec 초안 추출
  coverage                    커버리지 분석 실행

조회 명령:
  providers list|doctor       AI backend 목록 및 상태 확인
  agents list                 AGENTS.md 와 skills 확인
  trace <build-id>            build receipt / trace 확인
  explain <spec-id>           spec 의존성과 소유 파일 설명

플랫폼 명령:
  serve                       HTTP daemon 시작

개발 명령:
  parse-spec <file>           spec 파싱 결과를 JSON으로 출력
  version                     버전 출력

빠른 시작:
  1. plaincode init
  2. spec/_blueprint.md 를 복사해서 새 spec 작성
  3. plaincode build --spec <id>
  4. plaincode test --spec <id>
  5. plaincode run --spec <id> --build

참고:
  plaincode build 는 코드를 생성만 하고 서버를 자동 실행하지 않습니다.
  명세 검증은 plaincode test, 서버 관리는 plaincode run / status / stop 으로 합니다.

언어 선택:
  plaincode help --lang ko
  plaincode help --lang en

`
	default:
		return `plaincode — spec-first multi-agent build orchestrator

Usage: plaincode <command> [options]
Help:  plaincode help [--lang ko|en]

Core Commands:
  init                        Initialize a new PlainCode project
  build [--spec <id>]         Build specs into code
  test --spec <id>            Verify implementation against the spec
  run --spec <id>             Start a built service
  stop --spec <id>            Stop a managed service
  status [--spec <id>]        Show managed service status
  logs --spec <id>            Show stored runtime logs or events
  change -m "description"     Fix implementation bug (not spec change)
  takeover <file|package>     Extract spec from existing code
  coverage                    Run coverage analysis and gap filling

Inspection Commands:
  providers list|doctor       Manage AI backends
  agents list                 List AGENTS.md and skills
  trace <build-id>            Inspect build receipt and trace
  explain <spec-id>           Explain spec dependencies and ownership

Platform Commands:
  serve                       Start HTTP daemon (OpenAPI + SSE)

Development Commands:
  parse-spec <file>           Parse and dump a spec file (debug)
  version                     Print version

Quick Start:
  1. plaincode init
  2. Copy spec/_blueprint.md into a real spec file
  3. plaincode build --spec <id>
  4. plaincode test --spec <id>
  5. plaincode run --spec <id> --build

Note:
  plaincode build only generates code and receipts.
  Spec verification lives in plaincode test, and runtime management lives in plaincode run / status / stop.

Language:
  plaincode help --lang ko
  plaincode help --lang en

`
	}
}

func resolveHelpLanguage(args []string) string {
	fs := flag.NewFlagSet("help", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	lang := fs.String("lang", "", "Help language: ko or en")
	fs.StringVar(lang, "l", "", "Help language: ko or en")
	_ = fs.Parse(args)

	if normalized := normalizeLanguage(*lang); normalized != "" {
		return normalized
	}

	for _, key := range []string{"PLAINCODE_LANG", "LC_ALL", "LC_MESSAGES", "LANG"} {
		if normalized := normalizeLanguage(os.Getenv(key)); normalized != "" {
			return normalized
		}
	}

	return "en"
}

func normalizeLanguage(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(value, "ko"):
		return "ko"
	case strings.HasPrefix(value, "en"):
		return "en"
	default:
		return ""
	}
}
