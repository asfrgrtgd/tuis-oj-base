package core

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestBuildProblemTemplateZip(t *testing.T) {
	data, err := buildProblemTemplateZip()
	if err != nil {
		t.Fatalf("buildProblemTemplateZip error: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("zip reader error: %v", err)
	}

	expected := map[string]func(string) bool{
		"two-string/problem.yaml": func(s string) bool {
			return strings.Contains(s, `slug: two-string`) &&
				strings.Contains(s, `title: "Two String"`)
		},
		"two-string/statement.md": func(s string) bool {
			return strings.Contains(s, "## 問題文") &&
				strings.Contains(s, "文字列 S, T が与えられます") &&
				strings.Contains(s, "S と T を連結した文字列を 1 行で出力せよ")
		},
		"two-string/data/sample/01.in":  func(s string) bool { return s == "Hello\nOJ\n" },
		"two-string/data/sample/01.out": func(s string) bool { return s == "HelloOJ\n" },
		"two-string/data/secret/01.in":  func(s string) bool { return s == "abc\nxyz\n" },
		"two-string/data/secret/01.out": func(s string) bool { return s == "abcxyz\n" },
	}

	for _, f := range zr.File {
		verify, ok := expected[f.Name]
		if !ok {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(rc); err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
		rc.Close()
		if !verify(buf.String()) {
			t.Fatalf("content mismatch for %s", f.Name)
		}
		delete(expected, f.Name)
	}

	if len(expected) != 0 {
		t.Fatalf("missing files: %v", mapsKeys(expected))
	}
}

func mapsKeys(m map[string]func(string) bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
