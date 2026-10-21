package bench

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// The manifest is last: an interrupted write must not look complete.
var reportFileNames = [...]string{"results.json", "results.md", "evidence.json"}

func validateReportFiles(files map[string][]byte) error {
	if len(files) != len(reportFileNames) {
		return fmt.Errorf("report requires exactly results.json, results.md and evidence.json")
	}
	for _, name := range reportFileNames {
		if _, ok := files[name]; !ok {
			return fmt.Errorf("report missing %s", name)
		}
	}
	return nil
}

// writeReportFiles requires a trusted parent whose destination entry remains
// stable during initial creation/open. Portable os.Root has no atomic
// mkdir-and-open operation: replacement after Mkdir but before the first Lstat
// cannot be distinguished from the newly created directory. Once opened and
// identified, root-relative writes cannot be redirected by later path replacement.
func writeReportFiles(ctx context.Context, dir string, files map[string][]byte) error {
	return writeReportFilesWith(ctx, dir, files, writeReportFile)
}

// The writer seam permits deterministic write/close failures and directory
// replacement tests without process-wide filesystem hooks.
func writeReportFilesWith(ctx context.Context, dir string, files map[string][]byte, write func(context.Context, *os.Root, string, []byte) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateReportFiles(files); err != nil {
		return err
	}
	dir = filepath.Clean(dir)
	parent, err := os.OpenRoot(filepath.Dir(dir))
	if err != nil {
		return fmt.Errorf("open report parent: %w", err)
	}
	defer parent.Close()
	name := filepath.Base(dir)
	if err := parent.Mkdir(name, 0700); err != nil {
		return fmt.Errorf("claim new report directory: %w", err)
	}
	// This initial identity capture relies on the stable-parent precondition
	// above; it is not proof of ownership against a hostile concurrent rename.
	root, identity, err := openReportRoot(parent, name)
	if err != nil {
		return err
	}
	defer root.Close()
	for _, file := range reportFileNames {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := sameReportEntry(parent, name, identity); err != nil {
			return err
		}
		if err := write(ctx, root, file, files[file]); err != nil {
			return fmt.Errorf("write report %s: %w", file, err)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return sameReportEntry(parent, name, identity)
}

// openReportRoot rejects changes observed after its initial Lstat. It cannot
// establish the historical identity of an entry replaced before that observation.
func openReportRoot(parent *os.Root, name string) (*os.Root, os.FileInfo, error) {
	identity, err := parent.Lstat(name)
	if err != nil {
		return nil, nil, fmt.Errorf("inspect report directory: %w", err)
	}
	if !identity.IsDir() {
		return nil, nil, fmt.Errorf("report destination must be a directory without a symlink")
	}
	root, err := parent.OpenRoot(name)
	if err != nil {
		return nil, nil, fmt.Errorf("open report directory: %w", err)
	}
	opened, err := root.Stat(".")
	if err != nil || !os.SameFile(identity, opened) {
		root.Close()
		return nil, nil, fmt.Errorf("report directory changed while opening")
	}
	if err := sameReportEntry(parent, name, identity); err != nil {
		root.Close()
		return nil, nil, err
	}
	return root, identity, nil
}

func sameReportEntry(root *os.Root, name string, identity os.FileInfo) error {
	current, err := root.Lstat(name)
	if err != nil {
		return fmt.Errorf("inspect report entry %s: %w", name, err)
	}
	if current.Mode()&os.ModeSymlink != 0 || !os.SameFile(identity, current) {
		return fmt.Errorf("report entry %s changed", name)
	}
	return nil
}

func writeReportFile(ctx context.Context, root *os.Root, name string, data []byte) error {
	file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	identity, err := file.Stat()
	if err != nil {
		return errors.Join(err, file.Close())
	}
	if err := writeCloseReportFile(ctx, file, data); err != nil {
		return err
	}
	return sameReportEntry(root, name, identity)
}

func writeCloseReportFile(ctx context.Context, file io.WriteCloser, data []byte) (err error) {
	defer func() { err = errors.Join(err, file.Close()) }()
	for len(data) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		chunk := data[:min(len(data), 32<<10)]
		n, err := file.Write(chunk)
		if err != nil {
			return err
		}
		if n != len(chunk) {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return ctx.Err()
}

func checkReportFiles(ctx context.Context, dir string, files map[string][]byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateReportFiles(files); err != nil {
		return err
	}
	dir = filepath.Clean(dir)
	parent, err := os.OpenRoot(filepath.Dir(dir))
	if err != nil {
		return fmt.Errorf("open report parent: %w", err)
	}
	defer parent.Close()
	root, identity, err := openReportRoot(parent, filepath.Base(dir))
	if err != nil {
		return err
	}
	defer root.Close()
	for _, name := range reportFileNames {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := checkReportFile(ctx, root, name, files[name]); err != nil {
			return fmt.Errorf("check report %s: %w", name, err)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return sameReportEntry(parent, filepath.Base(dir), identity)
}

func checkReportFile(ctx context.Context, root *os.Root, name string, expected []byte) (err error) {
	inspected, err := root.Lstat(name)
	if err != nil {
		return err
	}
	if !inspected.Mode().IsRegular() {
		return fmt.Errorf("expected regular file without symlink")
	}
	file, err := root.OpenFile(name, bundleReadFlags, 0)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	opened, err := file.Stat()
	if err != nil {
		return err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(inspected, opened) {
		return fmt.Errorf("file changed while opening")
	}
	// The expected snapshot bounds reads even when the stored file is enormous.
	data, err := readBounded(file, int64(len(expected)))
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := sameReportEntry(root, name, opened); err != nil {
		return err
	}
	if !bytes.Equal(data, expected) {
		return fmt.Errorf("generated content differs")
	}
	return nil
}
