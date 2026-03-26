package speccheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	pruntime "github.com/bssm-oss/PlainCode/internal/runtime"
	"github.com/bssm-oss/PlainCode/internal/spec/ast"
	vtest "github.com/bssm-oss/PlainCode/internal/validate/test"
)

var (
	reGETPath            = regexp.MustCompile(`(?i)\bGET\s+(\S+)`)
	reJSONFragment       = regexp.MustCompile(`(\{.*\}|\[.*\])`)
	reFieldLengthKorean  = regexp.MustCompile(`(?i)^GET\s+(\S+).+?\s([A-Za-z_][A-Za-z0-9_]*)\s+길이(?:는)?\s+(-?\d+)`)
	reFieldLengthEnglish = regexp.MustCompile(`(?i)^GET\s+(\S+).+?\s([A-Za-z_][A-Za-z0-9_]*)\s+length(?:\s+is)?\s+(-?\d+)`)
	reFieldValueKorean   = regexp.MustCompile(`(?i)^GET\s+(\S+).+?\s([A-Za-z_][A-Za-z0-9_]*)\s+는\s+(.+?)\s+이다`)
	reFieldValueEnglish  = regexp.MustCompile(`(?i)^GET\s+(\S+).+?\s(?:has|where)\s+([A-Za-z_][A-Za-z0-9_]*)\s+(?:is\s+)?(.+)$`)
	reStatusCode         = regexp.MustCompile(`\b([1-5][0-9]{2})\b`)
)

// Options controls spec verification behavior.
type Options struct {
	RuntimeMode string
	WaitTimeout time.Duration
	SkipCommand bool
	KeepRunning bool
	HTTPTimeout time.Duration
}

// Result holds the outcome of spec-driven verification.
type Result struct {
	SpecID            string          `json:"spec_id"`
	Passed            bool            `json:"passed"`
	CommandResult     *vtest.Result   `json:"command_result,omitempty"`
	Oracles           []OracleResult  `json:"oracles,omitempty"`
	ParsedOracleCount int             `json:"parsed_oracle_count"`
	IgnoredOracles    []string        `json:"ignored_oracles,omitempty"`
	StartedRuntime    bool            `json:"started_runtime,omitempty"`
	RuntimeState      *pruntime.State `json:"runtime_state,omitempty"`
	Errors            []string        `json:"errors,omitempty"`
	DurationMS        int64           `json:"duration_ms"`
}

// OracleResult holds the outcome of a single parsed spec oracle.
type OracleResult struct {
	Raw          string `json:"raw"`
	Kind         string `json:"kind"`
	URL          string `json:"url"`
	Passed       bool   `json:"passed"`
	StatusCode   int    `json:"status_code,omitempty"`
	Error        string `json:"error,omitempty"`
	ResponseBody string `json:"response_body,omitempty"`
}

type oracle struct {
	Kind         string
	Raw          string
	Path         string
	ExpectStatus int
	ExpectJSON   string
	ExpectField  string
	ExpectValue  any
	ExpectLength *int
}

// Checker verifies an implementation against a parsed spec.
type Checker struct {
	runtime *pruntime.Manager
	tests   *vtest.Runner
	client  *http.Client
}

// New creates a spec checker.
func New(runtimeManager *pruntime.Manager) *Checker {
	return &Checker{
		runtime: runtimeManager,
		tests:   vtest.NewRunner(),
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

// Run executes tests.command and any parsed HTTP test oracles.
func (c *Checker) Run(ctx context.Context, spec *ast.Spec, projectDir string, opts Options) (*Result, error) {
	start := time.Now()
	result := &Result{SpecID: spec.ID}

	oracles, ignored := ParseHTTPOracles(spec.Body.TestOracles)
	result.ParsedOracleCount = len(oracles)
	result.IgnoredOracles = ignored

	if !opts.SkipCommand && strings.TrimSpace(spec.Tests.Command) != "" {
		commandResult, err := c.tests.Run(ctx, projectDir, spec.Tests.Command)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("running tests.command: %v", err))
		} else {
			result.CommandResult = commandResult
			if !commandResult.Passed {
				result.Errors = append(result.Errors, fmt.Sprintf("tests.command failed with exit code %d", commandResult.ExitCode))
			}
		}
	}

	if len(oracles) > 0 {
		var (
			runtimeState   *pruntime.State
			startedRuntime bool
		)

		if c.runtime == nil {
			result.Errors = append(result.Errors, "runtime manager is not configured")
		} else {
			existing, err := c.runtime.Status(ctx, spec.ID)
			if err == nil && existing.Status == pruntime.StatusRunning {
				runtimeState = existing
			} else {
				runtimeState, err = c.runtime.Start(ctx, spec, pruntime.StartOptions{
					Mode:          opts.RuntimeMode,
					HealthTimeout: opts.WaitTimeout,
				})
				if err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("starting runtime for spec oracles: %v", err))
				} else {
					startedRuntime = true
				}
			}
		}

		if runtimeState != nil {
			result.RuntimeState = runtimeState
			result.StartedRuntime = startedRuntime
			if startedRuntime && !opts.KeepRunning {
				defer func() {
					_, _ = c.runtime.Stop(context.Background(), spec.ID)
				}()
			}

			baseURL, err := baseURLFromState(runtimeState)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("deriving base URL for spec oracles: %v", err))
			} else {
				c.client.Timeout = timeoutOrDefault(opts.HTTPTimeout, 5*time.Second)
				result.Oracles = c.runHTTPOracles(baseURL, oracles)
				for _, item := range result.Oracles {
					if !item.Passed {
						result.Errors = append(result.Errors, fmt.Sprintf("oracle failed: %s", item.Raw))
					}
				}
			}
		}
	}

	if result.CommandResult == nil && len(oracles) == 0 {
		result.Errors = append(result.Errors, "spec has no executable tests: add tests.command or HTTP test oracles")
	}

	result.Passed = len(result.Errors) == 0
	result.DurationMS = time.Since(start).Milliseconds()
	return result, nil
}

// ParseHTTPOracles extracts runnable HTTP checks from the free-form Test oracles section.
func ParseHTTPOracles(text string) ([]oracle, []string) {
	lines := strings.Split(text, "\n")
	var parsed []oracle
	var ignored []string
	for _, line := range lines {
		raw := strings.TrimSpace(line)
		if raw == "" {
			continue
		}
		clean := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(raw, "-"), "*"), "•"))
		clean = strings.ReplaceAll(clean, "`", "")
		if !strings.Contains(strings.ToUpper(clean), "GET ") {
			ignored = append(ignored, raw)
			continue
		}

		if item, ok := parseFieldLengthOracle(raw, clean); ok {
			parsed = append(parsed, item)
			continue
		}
		if item, ok := parseFieldValueOracle(raw, clean); ok {
			parsed = append(parsed, item)
			continue
		}
		if item, ok := parseStatusOracle(raw, clean); ok {
			parsed = append(parsed, item)
			continue
		}
		ignored = append(ignored, raw)
	}
	return parsed, ignored
}

func parseFieldLengthOracle(raw, clean string) (oracle, bool) {
	for _, re := range []*regexp.Regexp{reFieldLengthKorean, reFieldLengthEnglish} {
		matches := re.FindStringSubmatch(clean)
		if len(matches) != 4 {
			continue
		}
		value, err := strconv.Atoi(matches[3])
		if err != nil {
			return oracle{}, false
		}
		return oracle{
			Kind:         "http_json_field_length",
			Raw:          raw,
			Path:         matches[1],
			ExpectField:  matches[2],
			ExpectLength: &value,
		}, true
	}
	return oracle{}, false
}

func parseFieldValueOracle(raw, clean string) (oracle, bool) {
	for _, re := range []*regexp.Regexp{reFieldValueKorean, reFieldValueEnglish} {
		matches := re.FindStringSubmatch(clean)
		if len(matches) != 4 {
			continue
		}
		value, err := parseScalar(matches[3])
		if err != nil {
			return oracle{}, false
		}
		return oracle{
			Kind:        "http_json_field_value",
			Raw:         raw,
			Path:        matches[1],
			ExpectField: matches[2],
			ExpectValue: value,
		}, true
	}
	return oracle{}, false
}

func parseStatusOracle(raw, clean string) (oracle, bool) {
	pathMatch := reGETPath.FindStringSubmatch(clean)
	statusMatch := reStatusCode.FindStringSubmatch(clean)
	if len(pathMatch) != 2 || len(statusMatch) != 2 {
		return oracle{}, false
	}
	status, err := strconv.Atoi(statusMatch[1])
	if err != nil {
		return oracle{}, false
	}
	item := oracle{
		Kind:         "http_status",
		Raw:          raw,
		Path:         pathMatch[1],
		ExpectStatus: status,
	}
	if jsonMatch := reJSONFragment.FindStringSubmatch(clean); len(jsonMatch) == 2 {
		item.Kind = "http_status_json"
		item.ExpectJSON = strings.TrimSpace(jsonMatch[1])
	}
	return item, true
}

func parseScalar(raw string) (any, error) {
	value := strings.TrimSpace(strings.Trim(raw, "."))
	value = strings.Trim(value, "\"'")
	if n, err := strconv.Atoi(value); err == nil {
		return n, nil
	}
	switch strings.ToLower(value) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	case "null":
		return nil, nil
	default:
		return value, nil
	}
}

func (c *Checker) runHTTPOracles(baseURL string, oracles []oracle) []OracleResult {
	results := make([]OracleResult, 0, len(oracles))
	for _, item := range oracles {
		results = append(results, c.runHTTPOracle(baseURL, item))
	}
	return results
}

func (c *Checker) runHTTPOracle(baseURL string, item oracle) OracleResult {
	targetURL := resolveTargetURL(baseURL, item.Path)
	result := OracleResult{
		Raw:  item.Raw,
		Kind: item.Kind,
		URL:  targetURL,
	}

	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	resp, err := c.client.Do(req)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	result.StatusCode = resp.StatusCode
	result.ResponseBody = strings.TrimSpace(string(body))

	if item.ExpectStatus != 0 && resp.StatusCode != item.ExpectStatus {
		result.Error = fmt.Sprintf("expected status %d, got %d", item.ExpectStatus, resp.StatusCode)
		return result
	}

	switch item.Kind {
	case "http_status":
		result.Passed = true
		return result
	case "http_status_json":
		if ok, msg := jsonEquivalent(result.ResponseBody, item.ExpectJSON); !ok {
			result.Error = msg
			return result
		}
	case "http_json_field_value":
		value, err := extractTopLevelField(result.ResponseBody, item.ExpectField)
		if err != nil {
			result.Error = err.Error()
			return result
		}
		if !reflect.DeepEqual(value, item.ExpectValue) {
			result.Error = fmt.Sprintf("expected %s=%v, got %v", item.ExpectField, item.ExpectValue, value)
			return result
		}
	case "http_json_field_length":
		value, err := extractTopLevelField(result.ResponseBody, item.ExpectField)
		if err != nil {
			result.Error = err.Error()
			return result
		}
		length, err := lengthOf(value)
		if err != nil {
			result.Error = err.Error()
			return result
		}
		if item.ExpectLength != nil && length != *item.ExpectLength {
			result.Error = fmt.Sprintf("expected len(%s)=%d, got %d", item.ExpectField, *item.ExpectLength, length)
			return result
		}
	}

	result.Passed = true
	return result
}

func baseURLFromState(state *pruntime.State) (string, error) {
	raw := strings.TrimSpace(state.HealthcheckURL)
	if raw == "" {
		return "", fmt.Errorf("runtime state does not have a healthcheck URL")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid healthcheck URL: %s", raw)
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

func resolveTargetURL(baseURL, target string) string {
	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return target
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(target, "/")
}

func jsonEquivalent(actual, expected string) (bool, string) {
	var actualValue any
	if err := json.Unmarshal([]byte(actual), &actualValue); err != nil {
		return false, fmt.Sprintf("response is not valid JSON: %v", err)
	}
	var expectedValue any
	if err := json.Unmarshal([]byte(expected), &expectedValue); err != nil {
		return false, fmt.Sprintf("expected JSON is invalid: %v", err)
	}
	if !reflect.DeepEqual(actualValue, expectedValue) {
		return false, fmt.Sprintf("expected JSON %s, got %s", expected, actual)
	}
	return true, ""
}

func extractTopLevelField(body, field string) (any, error) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return nil, fmt.Errorf("response is not valid JSON: %w", err)
	}
	value, ok := payload[field]
	if !ok {
		return nil, fmt.Errorf("response does not contain field %q", field)
	}
	return normalizeJSONNumber(value), nil
}

func normalizeJSONNumber(value any) any {
	switch v := value.(type) {
	case float64:
		if v == float64(int(v)) {
			return int(v)
		}
	}
	return value
}

func lengthOf(value any) (int, error) {
	switch v := value.(type) {
	case []any:
		return len(v), nil
	case string:
		return len(v), nil
	case map[string]any:
		return len(v), nil
	default:
		return 0, fmt.Errorf("field does not have a measurable length")
	}
}

func timeoutOrDefault(value, fallback time.Duration) time.Duration {
	if value <= 0 {
		return fallback
	}
	return value
}
