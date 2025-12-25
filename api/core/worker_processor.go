package core

import (
	"context"
	"errors"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// WorkerProcessor consumes submission IDs and runs judge.
type WorkerProcessor struct {
	subRepo            SubmissionRepository
	problemRepo        ProblemRepository
	judge              JudgeClient
	compileTimeLimitMs int
}

const defaultCompileTimeLimitMs = 5000

func NewWorkerProcessor(subRepo SubmissionRepository, problemRepo ProblemRepository, judge JudgeClient, compileTimeLimitMs int) *WorkerProcessor {
	if compileTimeLimitMs <= 0 {
		compileTimeLimitMs = defaultCompileTimeLimitMs
	}
	return &WorkerProcessor{
		subRepo:            subRepo,
		problemRepo:        problemRepo,
		judge:              judge,
		compileTimeLimitMs: compileTimeLimitMs,
	}
}

// Process takes a submission ID (as string from queue) and executes judge pipeline.
// Returns final verdict and a system-level error (non-nil when the job should be retried).
func (p *WorkerProcessor) Process(ctx context.Context, jobID string) (string, error) {
	id, err := strconv.ParseInt(jobID, 10, 64)
	if err != nil {
		return "", err
	}

	sub, err := p.subRepo.AcquirePending(ctx, id)
	if err != nil {
		return "", err
	}

	// Read source
	sourceBytes, err := os.ReadFile(sub.SourcePath)
	if err != nil {
		return "", err
	}

	// Problem limits / checker (fallback to defaults if missing)
	timeLimitMs := 2000
	memoryLimitMb := 256
	checkerType := "exact"
	checkerEps := 0.0
	if detail, err := p.problemRepo.FindDetail(ctx, sub.ProblemID); err == nil {
		if detail.TimeLimitMS > 0 {
			timeLimitMs = int(detail.TimeLimitMS)
		}
		if detail.MemoryLimitKB > 0 {
			// ceil KB -> MB
			memoryLimitMb = int((detail.MemoryLimitKB + 1023) / 1024)
			if memoryLimitMb == 0 {
				memoryLimitMb = 1
			}
		}
		if strings.TrimSpace(detail.CheckerType) != "" {
			checkerType = strings.ToLower(strings.TrimSpace(detail.CheckerType))
			checkerEps = detail.CheckerEps
		}
	}

	// Compile
	compileRes, _, artifactID, err := p.judge.Compile(ctx, sub.Language, string(sourceBytes), p.compileTimeLimitMs, memoryLimitMb)
	compileStdoutPath, compileStderrPath := "", ""
	if compileRes != nil {
		dir := filepath.Dir(sub.SourcePath)
		if out, ok := compileRes.Files["stdout"]; ok {
			compileStdoutPath, _ = writeFileContent(dir, "compile_stdout.txt", out)
		}
		if errOut, ok := compileRes.Files["stderr"]; ok {
			compileStderrPath, _ = writeFileContent(dir, "compile_stderr.txt", errOut)
		}
	}

	// If compile failed or errored
	if err != nil {
		return "", err
	}
	if compileRes.Status != "Accepted" || compileRes.ExitStatus != 0 {
		result := SubmissionResult{
			SubmissionID: sub.ID,
			Verdict:      "CE",
			StdoutPath:   stringPtrIfNotEmpty(compileStdoutPath),
			StderrPath:   stringPtrIfNotEmpty(compileStderrPath),
		}
		if compileRes != nil {
			if compileRes.Time > 0 {
				t := int32(compileRes.Time / 1_000_000)
				result.TimeMS = &t
			}
			if compileRes.Memory > 0 {
				m := int32(compileRes.Memory / 1024)
				result.MemoryKB = &m
			}
			if compileRes.Error != "" {
				result.ErrorMessage = ptr(compileRes.Error)
			}
		}
		if saveErr := p.subRepo.SaveResult(ctx, result, "failed"); saveErr != nil {
			log.Printf("failed to save compile result for %d: %v", id, saveErr)
		}
		return "CE", nil
	}

	// Run with artifact
	testCases, err := p.loadTestCases(ctx, sub.ProblemID)
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(sub.SourcePath)
	finalVerdict := "AC"
	finalStatus := "succeeded"
	runStdoutPath, runStderrPath := "", ""
	var finalTimeMS, finalMemKB *int32
	var finalExit *int32
	var finalErrMsg *string
	var details []SubmissionJudgeDetail

	for _, tc := range testCases {
		runRes, runErr := p.judge.RunWithArtifact(ctx, sub.Language, artifactID, tc.stdin, timeLimitMs, memoryLimitMb)

		verdict := mapVerdict(runRes)
		if verdict == "AC" {
			actualOut := ""
			if runRes != nil {
				actualOut = runRes.Files["stdout"]
			}
			if !outputsEqualWithChecker(actualOut, tc.expected, checkerType, checkerEps) {
				verdict = "WA"
			}
		}
		if runErr != nil {
			return "", runErr
		}

		// Track per-testcase detail and aggregate max time/memory
		detail := SubmissionJudgeDetail{Testcase: tc.name, Status: verdict}
		if runRes != nil {
			if runRes.Time > 0 {
				t := int32(runRes.Time / 1_000_000)
				detail.TimeMS = &t
				if finalTimeMS == nil || t > *finalTimeMS {
					tCopy := t
					finalTimeMS = &tCopy
				}
			}
			if runRes.Memory > 0 {
				m := int32(runRes.Memory / 1024)
				detail.MemoryKB = &m
				if finalMemKB == nil || m > *finalMemKB {
					mCopy := m
					finalMemKB = &mCopy
				}
			}
		}
		details = append(details, detail)

		// Capture first failing stdout/stderr for inspection
		if verdict != "AC" && finalVerdict == "AC" {
			if runRes != nil {
				if out, ok := runRes.Files["stdout"]; ok {
					runStdoutPath, _ = writeFileContent(dir, "run_stdout.txt", out)
				}
				if errOut, ok := runRes.Files["stderr"]; ok {
					runStderrPath, _ = writeFileContent(dir, "run_stderr.txt", errOut)
				}
				if runRes.ExitStatus != 0 {
					e := int32(runRes.ExitStatus)
					finalExit = &e
				}
				if runRes.Error != "" {
					finalErrMsg = ptr(runRes.Error)
				}
			}
			if runErr != nil && finalErrMsg == nil {
				finalErrMsg = ptr(runErr.Error())
			}
		}

		if verdict != "AC" {
			finalVerdict = verdict
			finalStatus = "failed"
			break
		}
	}

	result := SubmissionResult{
		SubmissionID: sub.ID,
		Verdict:      finalVerdict,
		StdoutPath:   stringPtrIfNotEmpty(runStdoutPath),
		StderrPath:   stringPtrIfNotEmpty(runStderrPath),
		TimeMS:       finalTimeMS,
		MemoryKB:     finalMemKB,
		ExitCode:     finalExit,
		ErrorMessage: finalErrMsg,
		Details:      details,
	}

	if saveErr := p.subRepo.SaveResult(ctx, result, finalStatus); saveErr != nil {
		log.Printf("failed to save run result for %d: %v", id, saveErr)
	}

	// Best effort artifact cleanup
	_ = p.judge.RemoveFiles(ctx, artifactID)

	return finalVerdict, nil
}

func mapVerdict(res *judgeResponse) string {
	if res == nil {
		return "RE"
	}
	switch res.Status {
	case "Accepted":
		if res.ExitStatus == 0 {
			return "AC"
		}
		return "RE"
	case "Time Limit Exceeded":
		return "TLE"
	case "Memory Limit Exceeded":
		return "MLE"
	case "Output Limit Exceeded":
		return "OLE"
	case "Runtime Error":
		return "RE"
	default:
		return "RE"
	}
}

func stringPtrIfNotEmpty(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

// testCase represents single input/output pair.
type testCase struct {
	name     string
	stdin    string
	expected string
}

// loadTestCases uses inline DB contents only (file path fallback is disabled).
func (p *WorkerProcessor) loadTestCases(ctx context.Context, problemID int64) ([]testCase, error) {
	dbCases, err := p.problemRepo.ListTestcases(ctx, problemID)
	if err != nil {
		return nil, err
	}
	if len(dbCases) == 0 {
		return nil, errors.New("no testcases defined for problem")
	}
	out := make([]testCase, 0, len(dbCases))
	for i, tc := range dbCases {
		out = append(out, testCase{
			name:     strconv.Itoa(i + 1),
			stdin:    tc.InputText,
			expected: tc.OutputText,
		})
	}
	return out, nil
}

func outputsEqualWithChecker(actual, expected, checkerType string, eps float64) bool {
	switch strings.ToLower(strings.TrimSpace(checkerType)) {
	case "eps":
		aa := strings.Fields(actual)
		bb := strings.Fields(expected)
		if len(aa) != len(bb) {
			return false
		}
		for i := range aa {
			x, err1 := strconv.ParseFloat(aa[i], 64)
			y, err2 := strconv.ParseFloat(bb[i], 64)
			if err1 != nil || err2 != nil {
				return false
			}
			if math.Abs(x-y) > eps {
				return false
			}
		}
		return true
	default:
		return strings.TrimRight(actual, "\r\n ") == strings.TrimRight(expected, "\r\n ")
	}
}
