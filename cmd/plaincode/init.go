package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bssm-oss/PlainCode/internal/config"
)

const (
	initBlueprintPath = "spec/_blueprint.md"
	initReadmePath    = "README.plaincode.ko.md"
)

const initBlueprintTemplate = `# PlainCode 청사진 템플릿

이 파일은 바로 build되는 spec이 아니라, 새 spec를 만들 때 복사해서 쓰는 템플릿입니다.

추천 사용 순서:
1. 이 파일을 spec/my-feature.md 로 복사합니다.
2. frontmatter의 id 와 managed_files.owned 를 실제 값으로 바꿉니다.
3. 본문을 요구사항에 맞게 채웁니다.
4. plaincode build --spec <id> 를 실행합니다.
5. plaincode test --spec <id> 로 명세 검증을 실행합니다.

예시:

---
id: example/feature
language: go
managed_files:
  owned:
    - main.go
    - main_test.go
backend:
  preferred:
    - cli:codex
approval: workspace-auto
tests:
  command: go test ./...
coverage:
  target: 0.50
budget:
  max_turns: 8
  max_cost_usd: 5
runtime:
  default_mode: process
  process:
    command: go run .
    working_dir: .
    healthcheck_url: http://127.0.0.1:8080/health
  docker:
    dockerfile: Dockerfile
    context: .
    ports:
      - 8080:8080
    healthcheck_url: http://127.0.0.1:8080/health
---
# Purpose

이 spec가 최종적으로 무엇을 만들지 한 줄로 적습니다.

## Functional behavior

- 사용자가 기대하는 핵심 동작을 bullet로 적습니다.
- HTTP API라면 요청, 응답, 상태 코드를 적습니다.
- UI라면 화면 상태와 사용자 액션을 적습니다.

## Inputs / outputs

- 입력 데이터, 환경 변수, 외부 의존성을 적습니다.
- 출력 파일, 응답 형식, 부수효과를 적습니다.

## Invariants

- 항상 유지되어야 하는 규칙을 적습니다.
- 예: JSON 응답 형식, 정렬 순서, 접근 제어 규칙

## Error cases

- 잘못된 입력이나 예외 상황에서 어떻게 동작해야 하는지 적습니다.

## Integration points

- 연결되는 파일, 패키지, API, Docker, 환경 변수 등을 적습니다.

## Test oracles

- 무엇이 통과하면 이 spec가 맞다고 볼지 적습니다.
- 예: go test ./... 통과
- 예: GET /health 가 200 과 {"status":"good"} 반환
- 예: GET /api/items 의 count 는 3 이다
- 예: GET /api/items 의 items 길이는 3 이다
`

const initReadmeTemplate = `# PlainCode 시작 가이드

이 디렉터리는 plaincode init 으로 만들어졌습니다.

## 먼저 보면 좋은 파일

- plaincode.yaml: 프로젝트 기본 설정
- spec/_blueprint.md: 새 spec를 만들 때 복사해서 쓰는 청사진 템플릿
- .plaincode/: build receipt와 상태가 저장되는 디렉터리
- .plaincode/runs/: 실행 중인 서비스 상태가 저장되는 디렉터리
- .plaincode/runs/*.log, *.events.jsonl: 실행 로그와 이벤트 타임라인

## 가장 빠른 시작 방법

1. 템플릿 복사

    cp spec/_blueprint.md spec/hello.md

2. spec/hello.md 안의 id, owned files, 요구사항 본문을 수정

3. 빌드 실행

    plaincode build --spec hello

4. 서비스 실행

    plaincode run --spec hello --build

5. 상태 확인 / 중지

    plaincode status --spec hello
    plaincode stop --spec hello

## 스펙을 쓸 때 핵심 규칙

- id 는 고유해야 합니다.
- managed_files.owned 에는 PlainCode가 책임질 파일만 넣습니다.
- spec 본문에는 구현 세부보다 **동작, 제약, 테스트 기준**을 명확히 적는 것이 중요합니다.
- 템플릿 파일 spec/_blueprint.md 는 그대로 두고, 항상 복사본으로 작업하는 것을 권장합니다.
- 런타임 관리는 spec 의 runtime 블록과 plaincode run/stop/status 명령으로 합니다.
- 명세 검증은 plaincode test 가 tests.command 와 Test oracles 를 함께 실행합니다.
- plaincode build 는 코드를 생성할 뿐 자동으로 서버를 켜지 않습니다.
- 서버 시작과 종료는 plaincode run, plaincode status, plaincode stop 으로 관리합니다.
- 디버깅이 필요하면 plaincode logs --spec <id> 와 .plaincode/runs/*.events.jsonl 을 봅니다.

## 자주 쓰는 명령어

    plaincode providers list
    plaincode build --spec hello
    plaincode build --spec hello --dry-run
    plaincode test --spec hello
    plaincode run --spec hello --build
    plaincode status --spec hello
    plaincode stop --spec hello
    plaincode logs --spec hello
    plaincode parse-spec spec/hello.md

## 기본 설정

현재 plaincode.yaml 은 cli:codex 를 기본 backend로 등록해 둡니다.
다른 CLI를 쓰고 싶으면 providers 와 defaults.backend 를 바꾸면 됩니다.

## 추천 흐름

1. 작은 spec 하나로 시작합니다.
2. owned 파일 범위를 작게 잡습니다.
3. build 결과를 확인하면서 spec를 점진적으로 구체화합니다.
4. 안정화되면 테스트와 coverage 목표를 올립니다.
`

func initProject(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "plaincode.yaml")); err == nil {
		return fmt.Errorf("plaincode.yaml already exists in this directory")
	}

	dirs := []string{
		"spec",
		".plaincode",
		".plaincode/builds",
		".plaincode/runs",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	if err := config.WriteDefault(dir); err != nil {
		return fmt.Errorf("writing plaincode.yaml: %w", err)
	}

	if err := writeInitFile(dir, initBlueprintPath, initBlueprintTemplate); err != nil {
		return err
	}
	if err := writeInitFile(dir, initReadmePath, initReadmeTemplate); err != nil {
		return err
	}

	return nil
}

func writeInitFile(dir, relPath, content string) error {
	path := filepath.Join(dir, relPath)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", relPath, err)
	}
	return nil
}
