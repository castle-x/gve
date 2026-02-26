package asset

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyAsset copies specified files from srcDir to destDir.
// It creates necessary subdirectories and preserves the relative path structure.
func CopyAsset(srcDir, destDir string, files []string) error {
	for _, f := range files {
		src := filepath.Join(srcDir, f)
		dst := filepath.Join(destDir, f)

		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf("create dir for %s: %w", f, err)
		}

		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("copy %s: %w", f, err)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
