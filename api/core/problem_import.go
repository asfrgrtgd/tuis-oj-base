package core

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	maxArchiveEntries   = 200
	maxArchiveTotalSize = 32 * 1024 * 1024
	maxArchiveFileSize  = 4 * 1024 * 1024
)

// ParseProblemArchive converts a zip problem package into inline DTO.
// Expected layout (トップフォルダは任意):
//
//	problem.yaml (required)
//	statement.md (required)
//	data/sample/*.in, *.out (optional, is_sample=true)
//	data/secret/*.in, *.out (optional, is_sample=false)
//
// Files may be placed directly under the archive root or under a single
// top-level folder whose name equals slug.
func ParseProblemArchive(data []byte) (ProblemCreateInput, error) {
	if len(data) == 0 {
		return ProblemCreateInput{}, errors.New("アーカイブが空です")
	}

	files := map[string][]byte{}
	// Accept zip only
	if len(data) < 4 || !bytes.Equal(data[:4], []byte{'P', 'K', 0x03, 0x04}) {
		return ProblemCreateInput{}, errors.New("zip 形式のみ対応しています")
	}
	rootName, err := collectFromZip(data, files)
	if err != nil {
		return ProblemCreateInput{}, err
	}

	if rootName == "" {
		return ProblemCreateInput{}, errors.New("zip のトップフォルダが必要です (slug と一致させてください)")
	}

	if len(files) == 0 {
		return ProblemCreateInput{}, errors.New("有効なファイルがありません")
	}

	configBytes, ok := files["problem.yaml"]
	if !ok {
		// handle two-string/two-string/... (トップフォルダ二重) by剥がす
		if stripPrefix(files, normalizeSlug(rootName)+"/") {
			configBytes, ok = files["problem.yaml"]
		}
	}
	if !ok {
		return ProblemCreateInput{}, errors.New("problem.yaml が見つかりません")
	}

	doc, err := parseProblemYAML(configBytes)
	if err != nil {
		return ProblemCreateInput{}, err
	}

	slug := normalizeSlug(doc.Slug)
	if slug == "" {
		return ProblemCreateInput{}, errors.New("slug は必須です (英小文字・数字・ハイフンのみ)")
	}
	if slug != normalizeSlug(rootName) {
		return ProblemCreateInput{}, errors.New("zip のトップフォルダ名と slug が一致していません")
	}

	// handle nested slug/slug/... (二重フォルダを許容)
	stripSlugPrefix(files, slug)

	statement, ok := files["statement.md"]
	if !ok {
		return ProblemCreateInput{}, errors.New("statement.md が見つかりません")
	}
	if strings.TrimSpace(doc.Title) == "" {
		return ProblemCreateInput{}, errors.New("title は必須です")
	}

	if doc.Limits.TimeMS <= 0 {
		doc.Limits.TimeMS = 2000
	}
	if doc.Limits.MemoryMB <= 0 {
		doc.Limits.MemoryMB = 256
	}

	// Collect testcases from data/sample and data/secret
	caseBuckets := map[string]struct {
		in       string
		out      string
		isSample bool
	}{}

	for name, content := range files {
		if strings.HasPrefix(name, "data/sample/") && strings.HasSuffix(name, ".in") {
			key := strings.TrimSuffix(strings.TrimPrefix(name, "data/sample/"), ".in")
			bucket := caseBuckets["sample/"+key]
			bucket.in = string(content)
			bucket.isSample = true
			caseBuckets["sample/"+key] = bucket
		}
		if strings.HasPrefix(name, "data/sample/") && strings.HasSuffix(name, ".out") {
			key := strings.TrimSuffix(strings.TrimPrefix(name, "data/sample/"), ".out")
			bucket := caseBuckets["sample/"+key]
			bucket.out = string(content)
			bucket.isSample = true
			caseBuckets["sample/"+key] = bucket
		}
		if strings.HasPrefix(name, "data/secret/") && strings.HasSuffix(name, ".in") {
			key := strings.TrimSuffix(strings.TrimPrefix(name, "data/secret/"), ".in")
			bucket := caseBuckets["secret/"+key]
			bucket.in = string(content)
			caseBuckets["secret/"+key] = bucket
		}
		if strings.HasPrefix(name, "data/secret/") && strings.HasSuffix(name, ".out") {
			key := strings.TrimSuffix(strings.TrimPrefix(name, "data/secret/"), ".out")
			bucket := caseBuckets["secret/"+key]
			bucket.out = string(content)
			caseBuckets["secret/"+key] = bucket
		}
	}

	if len(caseBuckets) == 0 {
		return ProblemCreateInput{}, errors.New("testcases が含まれていません (data/sample または data/secret)")
	}

	var keys []string
	for k := range caseBuckets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var tcs []ProblemTestcaseInput
	for _, key := range keys {
		b := caseBuckets[key]
		if strings.TrimSpace(b.in) == "" || strings.TrimSpace(b.out) == "" {
			return ProblemCreateInput{}, fmt.Errorf("%s の .in/.out が揃っていません", key)
		}
		var inPath, outPath string
		if strings.HasPrefix(key, "sample/") {
			base := strings.TrimPrefix(key, "sample/")
			inPath = path.Join("data/sample", base+".in")
			outPath = path.Join("data/sample", base+".out")
		} else {
			base := strings.TrimPrefix(key, "secret/")
			inPath = path.Join("data/secret", base+".in")
			outPath = path.Join("data/secret", base+".out")
		}
		tcs = append(tcs, ProblemTestcaseInput{
			InputText:  b.in,
			OutputText: b.out,
			InputPath:  inPath,
			OutputPath: outPath,
			IsSample:   b.isSample,
		})
	}

	isPublic := true
	if doc.Visibility.Public != nil {
		isPublic = *doc.Visibility.Public
	}
	return ProblemCreateInput{
		Title:         strings.TrimSpace(doc.Title),
		Slug:          slug,
		StatementMD:   string(statement),
		StatementPath: nil,
		TimeLimitMS:   int32(doc.Limits.TimeMS),
		MemoryLimitKB: int32(doc.Limits.MemoryMB * 1024),
		IsPublic:      isPublic,
		CheckerType:   doc.Checker.Type,
		CheckerEps:    doc.Checker.Eps,
		Testcases:     tcs,
	}, nil
}

type problemDoc struct {
	Slug   string `yaml:"slug"`
	Title  string `yaml:"title"`
	Limits struct {
		TimeMS   int `yaml:"time_ms"`
		MemoryMB int `yaml:"memory_mb"`
	} `yaml:"limits"`
	Checker struct {
		Type string  `yaml:"type"`
		Eps  float64 `yaml:"eps"`
	} `yaml:"checker"`
	Visibility struct {
		Public *bool `yaml:"public"`
	} `yaml:"visibility"`
}

func parseProblemYAML(b []byte) (problemDoc, error) {
	var doc problemDoc
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return doc, fmt.Errorf("problem.yaml の形式が不正です: %w", err)
	}
	doc.Title = strings.TrimSpace(doc.Title)
	if doc.Checker.Type == "" {
		doc.Checker.Type = "exact"
	}
	doc.Checker.Type = strings.ToLower(strings.TrimSpace(doc.Checker.Type))
	if doc.Checker.Type != "exact" && doc.Checker.Type != "eps" {
		return doc, fmt.Errorf("checker.type は exact または eps で指定してください")
	}
	if doc.Checker.Type == "eps" {
		if doc.Checker.Eps <= 0 {
			return doc, fmt.Errorf("checker.eps は 0 より大きい値を指定してください")
		}
	} else {
		doc.Checker.Eps = 0
	}
	return doc, nil
}

// collectFromZip reads zip entries into files map with size/entry/path validation.
func collectFromZip(data []byte, files map[string][]byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("zip を展開できません: %w", err)
	}
	var total int64
	hasRootLevel := false
	dirRoots := map[string]struct{}{}
	type entry struct {
		name    string
		content []byte
	}
	var entries []entry

	for i, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if i+1 > maxArchiveEntries {
			return "", errors.New("エントリ数が多すぎます (200 上限)")
		}
		norm := normalizeArchivePath(f.Name)
		if strings.HasPrefix(norm, "/") || strings.Contains(norm, "../") {
			return "", errors.New("不正なパスが含まれています")
		}
		if f.UncompressedSize64 > maxArchiveFileSize {
			return "", fmt.Errorf("ファイル %s が大きすぎます (上限 %d bytes)", f.Name, maxArchiveFileSize)
		}
		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("%s を開けません: %w", f.Name, err)
		}
		content, err := io.ReadAll(io.LimitReader(rc, 4*1024*1024))
		rc.Close()
		if err != nil {
			return "", fmt.Errorf("%s の読み込みに失敗しました: %w", f.Name, err)
		}
		if int64(len(content)) > maxArchiveFileSize {
			return "", fmt.Errorf("ファイル %s が大きすぎます (上限 %d bytes)", f.Name, maxArchiveFileSize)
		}
		total += int64(len(content))
		if total > maxArchiveTotalSize {
			return "", errors.New("展開後サイズが大きすぎます (32MB 上限)")
		}
		entries = append(entries, entry{name: norm, content: content})
		parts := strings.Split(norm, "/")
		if len(parts) == 1 {
			hasRootLevel = true
		} else if len(parts) > 1 && parts[0] != "" {
			dirRoots[parts[0]] = struct{}{}
		}
	}
	if hasRootLevel {
		return "", errors.New("トップフォルダが必要です (slug と一致させてください)")
	}
	if len(dirRoots) == 0 {
		return "", errors.New("トップフォルダが見つかりません")
	}
	if len(dirRoots) > 1 {
		return "", errors.New("トップフォルダは1つにまとめてください")
	}
	var root string
	for k := range dirRoots {
		root = k
	}
	for _, e := range entries {
		name := e.name
		if root != "" && strings.HasPrefix(name, root+"/") {
			name = strings.TrimPrefix(name, root+"/")
		}
		if name == "" {
			continue
		}
		files[name] = e.content
	}
	return root, nil
}

// No tar handler: zip only for import.

func normalizeArchivePath(p string) string {
	cleaned := path.Clean(strings.ReplaceAll(p, "\\", "/"))
	cleaned = strings.TrimPrefix(cleaned, "./")
	cleaned = strings.TrimPrefix(cleaned, "/")
	return cleaned
}

func normalizeSlug(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	var b strings.Builder
	lastHyphen := false
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastHyphen = false
			continue
		}
		if r == '-' || r == '_' || r == ' ' {
			if !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	slug = strings.ReplaceAll(slug, "--", "-")
	return slug
}

// stripSlugPrefix trims leading "<slug>/" from all entries when they are nested twice.
// If problem.yaml is already at root, this is a no-op.
func stripSlugPrefix(files map[string][]byte, slug string) {
	prefix := slug + "/"
	if _, ok := files["problem.yaml"]; ok {
		return
	}
	if _, ok := files[prefix+"problem.yaml"]; !ok {
		return
	}
	stripPrefix(files, prefix)
}

// stripPrefix removes a common prefix from all entries if problem.yaml exists under that prefix.
// Returns true if the map was modified.
func stripPrefix(files map[string][]byte, prefix string) bool {
	if _, ok := files[prefix+"problem.yaml"]; !ok {
		return false
	}
	newFiles := make(map[string][]byte, len(files))
	for k, v := range files {
		if !strings.HasPrefix(k, prefix) {
			newFiles[k] = v
			continue
		}
		nk := strings.TrimPrefix(k, prefix)
		if nk != "" {
			newFiles[nk] = v
		}
	}
	for k, v := range newFiles {
		files[k] = v
	}
	return true
}
