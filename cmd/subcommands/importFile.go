package subcommands

import (
	"crypto/md5"  // #nosec
	"crypto/sha1" // #nosec
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/forensicanalysis/forensicstore"
)

func ImportFile() *cobra.Command {
	var files []string
	cmd := &cobra.Command{
		Use:   "import-file <forensicstore>...",
		Short: "Import files",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := RequireStore(cmd, args); err != nil {
				return err
			}
			return cmd.MarkFlagRequired("file")
		},
		RunE: func(_ *cobra.Command, args []string) error {
			for _, url := range args {
				if err := singleFileImport(url, files); err != nil {
					return err
				}
			}
			return nil
		},
	}
	AddOutputFlags(cmd)
	cmd.Flags().StringArrayVar(&files, "file", []string{}, "forensicstore")
	return cmd
}

func singleFileImport(url string, files []string) error {
	store, teardown, err := forensicstore.Open(url)
	if err != nil {
		return err
	}
	defer teardown()

	for _, filePath := range files {
		err = filepath.Walk(filePath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			return insertFile(store, path)
		})

		if err != nil {
			return err
		}
	}
	return nil
}

func insertFile(store *forensicstore.ForensicStore, srcpath string) error {
	file := forensicstore.NewFile()
	file.Name = filepath.Base(srcpath)

	dstpath, storeFile, err := store.StoreFile(srcpath)
	if err != nil {
		return fmt.Errorf("error storing file: %w", err)
	}
	defer storeFile.Close()

	srcFile, err := os.Open(srcpath) // #nosec
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer srcFile.Close()

	size, hashes, err := hashCopy(storeFile, srcFile)
	if err != nil {
		return fmt.Errorf("error copying file: %w", err)
	}

	file.Size = float64(size)
	file.ExportPath = filepath.ToSlash(dstpath)
	file.Hashes = hashes

	_, err = store.InsertStruct(file)
	return err
}

func hashCopy(dst io.Writer, src io.Reader) (int64, map[string]interface{}, error) {
	md5hash, sha1hash, sha256hash := md5.New(), sha1.New(), sha256.New() // #nosec
	size, err := io.Copy(io.MultiWriter(dst, sha1hash, md5hash, sha256hash), src)
	if err != nil {
		return 0, nil, err
	}
	return size, map[string]interface{}{
		"MD5":     fmt.Sprintf("%x", md5hash.Sum(nil)),
		"SHA-1":   fmt.Sprintf("%x", sha1hash.Sum(nil)),
		"SHA-256": fmt.Sprintf("%x", sha256hash.Sum(nil)),
	}, nil
}