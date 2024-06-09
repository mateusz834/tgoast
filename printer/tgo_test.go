package printer

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mateusz834/tgoast/parser"
	"github.com/mateusz834/tgoast/scanner"
	"github.com/mateusz834/tgoast/token"
)

func TestTgoSyntax(t *testing.T) {
	const testdata = "./testdata/tgo"
	files, err := os.ReadDir(testdata)
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range files {
		ext := filepath.Ext(v.Name())
		if ext == ".tgo" {
			testFile := filepath.Join(testdata, v.Name())
			expectFileName := filepath.Join(testdata, v.Name()[:len(v.Name())-len(".tgo")]+".formatted")

			content, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatal(err)
			}

			fs := token.NewFileSet()
			f, err := parser.ParseFile(fs, filepath.Base(testFile), content, parser.SkipObjectResolution|parser.ParseComments)
			if err != nil {
				if v, ok := err.(scanner.ErrorList); ok {
					for _, err := range v {
						t.Errorf("%v", err)
					}
				}
				t.Errorf("Error while parsing file %v: %v", testFile, err)
				continue
			}

			var b strings.Builder
			var config = Config{
				Mode:     UseSpaces | TabIndent | Mode(normalizeNumbers),
				Tabwidth: 8,
			}
			if err := config.Fprint(&b, fs, f); err != nil {
				t.Fatal(err)
			}

			expect, err := os.ReadFile(expectFileName)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					if err := os.WriteFile(expectFileName, []byte(b.String()), 06660); err != nil {
						t.Fatal(err)
					}
					continue
				}
				t.Fatal(err)
			}

			got := b.String()
			if string(expect) != got {
				t.Errorf("unexpected in %v", testFile)
				d, err := gitDiff(t.TempDir(), string(expect), got)
				if err == nil {
					t.Logf("\n%v", d)
				}
			}
		}
	}
}

func gitDiff(tmpDir string, got, expect string) (string, error) {
	gotPath := filepath.Join(tmpDir, "got")
	gotFile, err := os.Create(gotPath)
	if err != nil {
		return "", err
	}
	defer gotFile.Close()
	if _, err := gotFile.WriteString(got); err != nil {
		return "", err
	}

	expectPath := filepath.Join(tmpDir, "expect")
	expectFile, err := os.Create(expectPath)
	if err != nil {
		return "", err
	}
	defer expectFile.Close()
	if _, err := expectFile.WriteString(expect); err != nil {
		return "", err
	}

	var out strings.Builder
	cmd := exec.Command("git", "diff", "-U 100000", "--no-index", "--color=always", "--ws-error-highlight=all", gotPath, expectPath)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil && cmd.ProcessState.ExitCode() != 1 {
		return "", err
	}
	return out.String(), nil
}
