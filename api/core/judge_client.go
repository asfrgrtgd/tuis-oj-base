package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// JudgeClient abstracts go-judge interaction.
type JudgeClient interface {
	Compile(ctx context.Context, lang, source string, timeLimitMs, memoryLimitMb int) (*judgeResponse, string, string, error)
	RunWithArtifact(ctx context.Context, lang, artifactID, stdin string, timeLimitMs, memoryLimitMb int) (*judgeResponse, error)
	RemoveFiles(ctx context.Context, ids ...string) error
}

// HTTPJudgeClient calls go-judge HTTP endpoints.
type HTTPJudgeClient struct {
	client *http.Client
	base   string
}

func NewHTTPJudgeClient(baseURL string) *HTTPJudgeClient {
	return &HTTPJudgeClient{
		client: &http.Client{Timeout: 30 * time.Second},
		base:   baseURL,
	}
}

// go-judge request payload structures

type judgeFile struct {
	Name    string  `json:"name,omitempty"`
	Max     int     `json:"max,omitempty"`
	Content *string `json:"content,omitempty"`
	FileID  string  `json:"fileId,omitempty"`
}

type judgeCommand struct {
	Args          []string             `json:"args"`
	Env           []string             `json:"env,omitempty"`
	Files         []judgeFile          `json:"files"`
	CPULimit      int64                `json:"cpuLimit"`
	MemoryLimit   int64                `json:"memoryLimit"`
	ProcLimit     int32                `json:"procLimit"`
	CopyIn        map[string]judgeFile `json:"copyIn,omitempty"`
	CopyOut       []string             `json:"copyOut,omitempty"`
	CopyOutCached []string             `json:"copyOutCached,omitempty"`
}

type judgeResponse struct {
	Status     string            `json:"status"`
	Time       int64             `json:"time"`
	Memory     int64             `json:"memory"`
	ExitStatus int               `json:"exitStatus"`
	Error      string            `json:"error"`
	Files      map[string]string `json:"files"`
	FileIDs    map[string]string `json:"fileIds"`
}

// JudgeResponse is an exported alias for test/mocking.
type JudgeResponse = judgeResponse

type judgeLangConfig struct {
	SourceName          string
	CompileArgs         []string
	CompileCopyOutCache []string
	ArtifactKey         string
	RunArgs             []string
}

var judgeLangConfigs = map[string]judgeLangConfig{
	"c": {
		SourceName:          "main.c",
		CompileArgs:         []string{"/usr/bin/gcc", "main.c", "-std=gnu17", "-O2", "-pipe", "-static", "-s", "-o", "main"},
		CompileCopyOutCache: []string{"main"},
		ArtifactKey:         "main",
		RunArgs:             []string{"./main"},
	},
	"cpp": {
		SourceName:          "main.cpp",
		CompileArgs:         []string{"/usr/bin/g++", "main.cpp", "-std=gnu++17", "-O2", "-pipe", "-s", "-o", "main"},
		CompileCopyOutCache: []string{"main"},
		ArtifactKey:         "main",
		RunArgs:             []string{"./main"},
	},
	"python": {
		SourceName:          "main.py",
		CompileArgs:         []string{"/usr/bin/python3", "-m", "py_compile", "main.py"},
		CompileCopyOutCache: []string{"main.py"},
		ArtifactKey:         "main.py",
		RunArgs:             []string{"/usr/bin/python3", "main.py"},
	},
	"java": {
		SourceName:          "Main.java",
		CompileArgs:         []string{"/bin/sh", "-c", "javac Main.java && jar cfe Main.jar Main *.class"},
		CompileCopyOutCache: []string{"Main.jar"},
		ArtifactKey:         "Main.jar",
		RunArgs:             []string{"/usr/bin/java", "-jar", "Main.jar"},
	},
}

func langConfigFor(key string) judgeLangConfig {
	k := strings.ToLower(strings.TrimSpace(key))
	if cfg, ok := judgeLangConfigs[k]; ok {
		return cfg
	}
	return judgeLangConfigs["c"]
}

// Compile builds source code and returns compile result plus cached artifact id (no run).
func (c *HTTPJudgeClient) Compile(ctx context.Context, lang, source string, timeLimitMs, memoryLimitMb int) (*judgeResponse, string, string, error) {
	if c.base == "" {
		return nil, "", "", errors.New("go-judge url not configured")
	}
	cfg := langConfigFor(lang)

	if timeLimitMs <= 0 {
		timeLimitMs = 2000
	}
	if memoryLimitMb <= 0 {
		memoryLimitMb = 256
	}
	cpuLimit := int64(timeLimitMs) * 1_000_000 // ms -> ns
	memLimit := int64(memoryLimitMb) * 1024 * 1024

	cmd := judgeCommand{
		Args:          cfg.CompileArgs,
		Env:           []string{"PATH=/usr/bin:/bin"},
		Files:         []judgeFile{{Name: "stdout", Max: 10240}, {Name: "stderr", Max: 10240}},
		CPULimit:      cpuLimit,
		MemoryLimit:   memLimit,
		ProcLimit:     50,
		CopyIn:        map[string]judgeFile{cfg.SourceName: {Content: &source}},
		CopyOutCached: cfg.CompileCopyOutCache,
	}

	payload := map[string]any{"cmd": []judgeCommand{cmd}}
	b, _ := json.Marshal(payload)
	log.Printf("judge compile lang=%s time_ms=%d mem_mb=%d size=%dB", lang, timeLimitMs, memoryLimitMb, len(source))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/run", bytes.NewReader(b))
	if err != nil {
		return nil, "", "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, "", "", err
	}
	defer resp.Body.Close()

	var body []judgeResponse
	if resp.StatusCode >= 300 {
		var textErr string
		_ = json.NewDecoder(resp.Body).Decode(&textErr)
		return nil, "", "", fmt.Errorf("judge returned status %d: %s", resp.StatusCode, textErr)
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, "", "", err
	}
	if len(body) == 0 {
		return nil, "", "", fmt.Errorf("empty judge response")
	}

	r := body[0]
	artifactID := ""
	if r.FileIDs != nil {
		artifactID = r.FileIDs[cfg.ArtifactKey]
	}

	return &r, cfg.ArtifactKey, artifactID, nil
}

// RunWithArtifact executes the compiled artifact with provided stdin.
func (c *HTTPJudgeClient) RunWithArtifact(ctx context.Context, lang, artifactID, stdin string, timeLimitMs, memoryLimitMb int) (*judgeResponse, error) {
	if c.base == "" {
		return nil, errors.New("go-judge url not configured")
	}
	if artifactID == "" {
		return nil, errors.New("empty artifact id")
	}
	cfg := langConfigFor(lang)

	if timeLimitMs <= 0 {
		timeLimitMs = 2000
	}
	if memoryLimitMb <= 0 {
		memoryLimitMb = 256
	}
	cpuLimit := int64(timeLimitMs) * 1_000_000
	memLimit := int64(memoryLimitMb) * 1024 * 1024

	// stdout/stderr を大きめに確保（ソートなど大出力系に対応）
	const stdoutLimit = 10_000_000 // 10MB
	files := []judgeFile{
		{Content: &stdin},
		{Name: "stdout", Max: stdoutLimit},
		{Name: "stderr", Max: 10240},
	}

	cmd := judgeCommand{
		Args:        cfg.RunArgs,
		Env:         []string{"PATH=/usr/bin:/bin"},
		Files:       files,
		CPULimit:    cpuLimit,
		MemoryLimit: memLimit,
		ProcLimit:   50,
		CopyIn: map[string]judgeFile{
			cfg.ArtifactKey: {FileID: artifactID},
		},
	}

	payload := map[string]any{"cmd": []judgeCommand{cmd}}
	b, _ := json.Marshal(payload)
	log.Printf("judge run lang=%s time_ms=%d mem_mb=%d stdin_bytes=%d", lang, timeLimitMs, memoryLimitMb, len(stdin))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/run", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body []judgeResponse
	if resp.StatusCode >= 300 {
		var textErr string
		_ = json.NewDecoder(resp.Body).Decode(&textErr)
		return nil, fmt.Errorf("judge returned status %d: %s", resp.StatusCode, textErr)
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("empty judge response")
	}
	return &body[0], nil
}

// RemoveFiles attempts to delete cached artifacts from go-judge (best-effort).
func (c *HTTPJudgeClient) RemoveFiles(ctx context.Context, ids ...string) error {
	if c.base == "" {
		return errors.New("go-judge url not configured")
	}
	if len(ids) == 0 {
		return nil
	}

	for _, id := range ids {
		if strings.TrimSpace(id) == "" {
			continue
		}
		endpoint := fmt.Sprintf("%s/file/%s", c.base, url.PathEscape(id))
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
		if err != nil {
			return err
		}
		resp, err := c.client.Do(req)
		if err != nil {
			return err
		}
		resp.Body.Close()

		if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("file delete returned status %d for id %s", resp.StatusCode, id)
		}
	}
	return nil
}

// Utility helpers

func ptr[T any](v T) *T { return &v }

func writeFileContent(dir, name, content string) (string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}
	return path, nil
}
