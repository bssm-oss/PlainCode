package coverage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCoverProfile(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, "unit.out")
	data := `mode: atomic
example.com/project/main.go:10.1,12.2 2 1
example.com/project/main.go:14.1,16.2 2 0
`
	if err := os.WriteFile(profile, []byte(data), 0644); err != nil {
		t.Fatalf("write profile: %v", err)
	}

	report, err := parseCoverProfile(profile)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if report.TotalLines != 4 || report.CoveredLines != 2 {
		t.Fatalf("unexpected totals: %+v", report)
	}
	if report.Percentage != 0.5 {
		t.Fatalf("unexpected percentage: %f", report.Percentage)
	}
	if len(report.Files["example.com/project/main.go"].UncoveredRanges) != 1 {
		t.Fatalf("expected uncovered range: %+v", report.Files)
	}
}
