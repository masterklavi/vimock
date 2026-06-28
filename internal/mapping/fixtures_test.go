package mapping

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCurrentFixtureMappings(t *testing.T) {
	roots := []string{
		filepath.Join("..", "..", "examples"),
		filepath.Join("..", "..", "autotests-example", "mocks"),
	}

	for _, root := range roots {
		if _, err := os.Stat(root); err != nil {
			t.Logf("skip missing fixtures root %s: %v", root, err)
			continue
		}

		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				return nil
			}

			t.Run(path, func(t *testing.T) {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read fixture: %v", err)
				}
				if _, err := ParseJSON(data); err != nil {
					t.Fatalf("parse mapping: %v", err)
				}
			})
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
}
