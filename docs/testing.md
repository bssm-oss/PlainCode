# Testing Documentation

## Overview

PlainCode has **9 test suites** with **30+ test cases** covering the entire pipeline from spec parsing to E2E build execution.

```bash
go test ./...
```

```
ok   internal/spec/parser         — 6 tests + 2 benchmarks
ok   internal/graph               — 5 tests
ok   internal/workspace/fsguard   — 8 tests
ok   internal/config              — 5 tests
ok   internal/receipt             — 3 tests
ok   internal/spec/ir             — 4 tests
ok   internal/backend/mock        — 5 tests
ok   internal/backend/cli         — 8 tests
ok   internal/app                 — 6 tests
```

---

## 1. Spec Parser Tests (`internal/spec/parser/`)

**파일**: `parser_test.go`, `parser_bench_test.go`

### TestParse_ValidSpec
- **입력**: 완전한 spec 문자열 (id, language, imports, managed_files, backend, approval, tests, coverage, budget + body sections)
- **검증**:
  - `spec.ID == "billing/invoice-pdf"`
  - `spec.Language == "go"`
  - `spec.Imports == ["billing/shared/money"]`
  - `spec.ManagedFiles.Owned` 길이 1
  - `spec.Approval == "workspace-auto"`
  - `spec.Coverage.Target == 0.85`
  - `spec.Budget.MaxTurns == 12`
  - `spec.Hash` 비어있지 않음 (SHA-256 계산됨)
  - `spec.Body.Purpose` 비어있지 않음
  - `spec.Body.FunctionalBehavior` 비어있지 않음
- **결과**: PASS — frontmatter 모든 필드 + body 섹션 정확히 파싱

### TestParse_MissingID
- **입력**: `id` 필드 없는 spec
- **검증**: 에러 반환됨
- **결과**: PASS — id는 필수 필드

### TestParse_MissingFrontmatter
- **입력**: `---` 구분자 없는 순수 마크다운
- **검증**: 에러 반환됨
- **결과**: PASS — frontmatter 없으면 파싱 거부

### TestParse_UnknownField
- **입력**: `unknown_field: bad` 포함된 spec
- **검증**: 에러 반환됨
- **결과**: PASS — `yaml.v3 KnownFields(true)` 가 알 수 없는 필드 거부

### TestSplitFrontmatter
- **입력**: `---\nid: test\n---\n# Body\nContent`
- **검증**: frontmatter와 body가 정확히 분리됨
- **결과**: PASS

### TestExtractSections
- **입력**: `## Purpose\nDo things.\n\n## Error cases\nNone.\n`
- **검증**: `sections["purpose"] == "Do things."`, `sections["error cases"] == "None."`
- **결과**: PASS — 헤딩 기반 섹션 추출 동작

### BenchmarkParse
- **입력**: 완전한 spec (frontmatter + 8 body sections)
- **결과**: **18,320 ns/op** (M4 Pro) — LLM 호출이 수십초인 파이프라인에서 무시 가능한 수준

### BenchmarkSplitFrontmatter
- **결과**: **338 ns/op** — 극도로 빠름

---

## 2. Build Graph Tests (`internal/graph/`)

**파일**: `graph_test.go`

### TestTopologicalSort_Simple
- **설정**: 3개 spec (a → b → c 의존)
- **검증**: 정렬 결과에서 a가 b보다 먼저, b가 c보다 먼저 나옴
- **결과**: PASS — 의존성 순서 보장

### TestTopologicalSort_CycleDetection
- **설정**: a → b, b → a (순환 의존)
- **검증**: 에러 반환됨
- **결과**: PASS — 순환 감지 및 거부

### TestMarkDirty_NewSpec
- **설정**: receipt 없는 새 spec
- **검증**: `IsDirty == true`
- **결과**: PASS — 한번도 빌드 안 된 spec은 dirty

### TestMarkDirty_UnchangedSpec
- **설정**: receipt hash와 현재 hash가 같은 spec
- **검증**: `IsDirty == false`
- **결과**: PASS — 변경 없으면 rebuild 안 함

### TestMarkDirty_Propagation
- **설정**: a가 dirty, b가 a에 의존
- **검증**: b도 `IsDirty == true`
- **결과**: PASS — 의존하는 spec이 dirty면 전파됨

---

## 3. File Ownership Tests (`internal/workspace/fsguard/`)

**파일**: `fsguard_test.go`

### TestClassify_Owned
- **설정**: spec-a가 `invoice.go`를 owned로 등록
- **검증**: spec-a 관점에서 Owned로 분류
- **결과**: PASS

### TestClassify_OwnedByOther
- **설정**: spec-a가 `invoice.go`를 소유
- **검증**: spec-b 관점에서 OwnedByOtherSpec으로 분류
- **결과**: PASS — 다른 spec의 파일 접근 감지

### TestClassify_Shared / Readonly / Unmanaged
- **결과**: 각각 PASS — 모든 분류 정확

### TestValidatePatch_ReadonlyRejected
- **검증**: readonly 파일 수정 시 에러 반환
- **결과**: PASS

### TestValidatePatch_OwnedByOtherRejected
- **검증**: 다른 spec 소유 파일 수정 시 에러 반환
- **결과**: PASS

### TestValidatePatch_OwnedAllowed / SharedAllowed
- **검증**: 자기 소유 파일과 shared 파일은 허용
- **결과**: PASS

---

## 4. Config Tests (`internal/config/`)

**파일**: `config_test.go`

### TestDefaultProjectConfig
- **검증**: 기본값 확인 (version=1, spec_dir="spec", state_dir=".plaincode", approval="patch")
- **결과**: PASS

### TestValidate_BadVersion / EmptySpecDir
- **검증**: version이 1이 아니거나 spec_dir이 비어있으면 에러
- **결과**: PASS

### TestWriteAndLoad
- **설정**: temp 디렉터리에 WriteDefault → Load
- **검증**: 저장된 config를 다시 읽어서 값 일치 확인
- **결과**: PASS — 직렬화/역직렬화 round-trip 동작

### TestLoad_MissingFile
- **검증**: plaincode.yaml 없으면 기본값 반환 (에러 아님)
- **결과**: PASS

---

## 5. Receipt Store Tests (`internal/receipt/`)

**파일**: `store_test.go`

### TestStore_SaveAndLoad
- **설정**: receipt 생성 → Save → Load
- **검증**: 모든 필드 (spec_id, status, tests_passed) 일치
- **결과**: PASS — JSON 직렬화 round-trip

### TestStore_SpecHashes
- **설정**: 2개 다른 spec의 receipt 저장
- **검증**: SpecHashes()가 각 spec의 마지막 hash 반환
- **결과**: PASS — dirty detection에 사용되는 hash map 정확

### TestStore_LoadMissing
- **검증**: 존재하지 않는 build ID로 Load 시 에러
- **결과**: PASS

---

## 6. Spec IR Tests (`internal/spec/ir/`)

**파일**: `resolver_test.go`

### TestResolve_Simple
- **설정**: spec a와 b (b가 a를 import)
- **검증**: b의 ResolvedImports에 a가 있음
- **결과**: PASS

### TestResolve_MissingImport
- **설정**: 존재하지 않는 spec을 import
- **검증**: 에러 반환
- **결과**: PASS

### TestResolve_OwnershipConflict
- **설정**: 두 spec이 같은 파일을 owned로 선언
- **검증**: 에러 반환 ("ownership conflict")
- **결과**: PASS — 빌드 시작 전에 충돌 감지

### TestResolve_NoConflict
- **설정**: 두 spec이 다른 파일을 소유
- **검증**: 에러 없음
- **결과**: PASS

---

## 7. Mock Backend Tests (`internal/backend/mock/`)

**파일**: `mock_test.go`

### TestMockBackend_ID / Capabilities
- **검증**: ID와 Capabilities 정확
- **결과**: PASS

### TestMockBackend_Execute
- **설정**: SetResponse로 응답 설정 → Execute
- **검증**: patches 1개, turns 1
- **결과**: PASS

### TestMockBackend_WithDelay
- **설정**: 10ms 지연 설정
- **검증**: 실행 시간 >= 10ms
- **결과**: PASS

### TestMockBackend_ContextCancel
- **설정**: 10ms 타임아웃 + 5초 지연
- **검증**: context cancellation 에러
- **결과**: PASS — 타임아웃 제대로 동작

---

## 8. CLI Adapter Tests (`internal/backend/cli/`)

**파일**: `cli_test.go`, `adapters_test.go`

### TestParseFileBlocks
- **입력**: 파일 블록 2개 포함된 텍스트 출력
- **검증**: WriteFile PatchOp 2개 추출, 경로와 내용 정확
- **결과**: PASS

### TestParseFileBlocks_Empty
- **입력**: 파일 블록 없는 일반 텍스트
- **검증**: 패치 0개
- **결과**: PASS

### TestAllAdapters_ID
- **검증**: 6개 어댑터 모두 올바른 ID 반환
  - `cli:claude`, `cli:codex`, `cli:gemini`, `cli:copilot`, `cli:cursor`, `cli:opencode`
- **결과**: PASS

### TestClaude_BuildArgs
- **검증**: 각 프로필에 대해 올바른 플래그 생성
  - plan → `--print`
  - patch → `--permission-mode`
  - full-trust → `--dangerously-skip-permissions`
- **결과**: PASS

### TestCodex_BuildArgs
- **검증**: plan → `read-only`, workspace-auto → `--full-auto`, full-trust → `--dangerously-bypass-approvals-and-sandbox`
- **결과**: PASS

### TestGemini_BuildArgs
- **검증**: full-trust → `--yolo`
- **결과**: PASS

### TestCopilot_BuildArgs
- **검증**: full-trust → `--yolo`
- **결과**: PASS

---

## 9. E2E Build Pipeline Tests (`internal/app/`)

**파일**: `builder_test.go`

### TestBuild_SingleSpec_Success
- **설정**: temp 프로젝트 + sample spec + mock backend
- **실행**: builder.Build(specID="hello/greeter", skipTests=true)
- **검증**:
  - status == "success"
  - specID == "hello/greeter"
  - receipt != nil
  - backendID == "mock:default"
  - specHash 비어있지 않음
  - `.plaincode/builds/<id>/receipt.json` 파일 존재
- **결과**: PASS — 전체 파이프라인 (parse → graph → backend → patch → receipt) E2E 동작

### TestBuild_DryRun
- **설정**: dry-run 모드
- **검증**: backend 실행 없이 success 반환
- **결과**: PASS

### TestBuild_NoDirtySpecs
- **설정**: 한 번 빌드 후 두 번째 빌드 (변경 없음)
- **검증**: 두 번째 빌드에서 dirty spec 없음 감지
- **결과**: PASS — hash 기반 dirty detection 동작

### TestBuild_SpecNotFound
- **설정**: 존재하지 않는 spec ID
- **검증**: 에러 반환
- **결과**: PASS

### TestBuild_OwnershipViolation
- **설정**: 두 spec이 같은 파일을 소유 (충돌)
- **검증**: 충돌 감지 및 적절한 처리
- **결과**: PASS

### TestLoadSpecs
- **설정**: temp 디렉터리에 spec 파일 생성
- **검증**: LoadSpecs가 1개 spec 발견, graph size 1
- **결과**: PASS — 디렉터리 스캔 동작

---

## Running Specific Tests

```bash
# 전체
go test ./...

# 특정 패키지
go test ./internal/spec/parser/

# 특정 테스트
go test ./internal/app/ -run TestBuild_SingleSpec_Success

# 벤치마크
go test -bench=. ./internal/spec/parser/

# 상세 출력
go test -v ./internal/app/
```
