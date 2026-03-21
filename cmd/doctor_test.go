package cmd

import (
	"bytes"
	"strings"
	"testing"

	"clashctl/internal/mihomo"
)

func TestBuildDoctorReportSummarizesResults(t *testing.T) {
	results := []mihomo.CheckResult{
		{Name: "binary", Passed: true},
		{Name: "controller", Passed: false, Problem: "connection refused"},
	}
	report := buildDoctorReport("doctor", true, results, []string{"use sudo"})

	if report.Command != "doctor" {
		t.Fatalf("Command = %q, want doctor", report.Command)
	}
	if !report.TunMode {
		t.Fatal("TunMode should be true")
	}
	if report.Summary.Passed != 1 || report.Summary.Failed != 1 {
		t.Fatalf("summary = %#v, want 1 passed and 1 failed", report.Summary)
	}
	if len(report.Hints) != 1 || report.Hints[0] != "use sudo" {
		t.Fatalf("Hints = %#v", report.Hints)
	}
}

func TestPrintDoctorResultsIncludesHintsAndErrorSummary(t *testing.T) {
	report := buildDoctorReport("doctor openai", false, []mihomo.CheckResult{{
		Name:    "controller",
		Passed:  false,
		Problem: "timeout",
		Suggest: "check proxy",
	}}, []string{"switch node"})

	var buf bytes.Buffer
	err := printDoctorResults(&buf, report)
	if err == nil {
		t.Fatal("printDoctorResults() should return an error when checks fail")
	}
	if got := err.Error(); got != "存在 1 项检查未通过" {
		t.Fatalf("error = %q", got)
	}

	out := buf.String()
	for _, want := range []string{"问题: timeout", "建议: check proxy", "结论:", "switch node", "检查完成: 0 通过, 1 失败"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q in:\n%s", want, out)
		}
	}
}
