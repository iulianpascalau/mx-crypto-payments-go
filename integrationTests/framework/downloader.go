package framework

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	ContractCreditsVersionTag = "v1.0.0"
	ContractCreditsURL        = "https://github.com/iulianpascalau/credits-contract-rs/releases/download/" + ContractCreditsVersionTag + "/credits-contract.zip"
)

// EnsureContractCredits will fetch the provided contract release artifact and unzip it
func EnsureContractCredits(contractURL string, targetDir string) error {
	contractDir := filepath.Join(targetDir, "credits")
	contractPath := filepath.Join(contractDir, "credits.wasm")

	if _, err := os.Stat(contractPath); err == nil {
		// Contract exists. We could check versioning (e.g. hash) but for now existence matches intent.
		return nil
	}

	log.Info(fmt.Sprintf("Downloading credits contract from %s...", contractURL))
	resp, err := http.Get(contractURL)
	if err != nil {
		return fmt.Errorf("failed to download contract: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "credits-contract-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	_ = tmpFile.Close()

	// Unzip
	r, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open zip reader: %w", err)
	}
	defer func() {
		_ = r.Close()
	}()

	err = os.MkdirAll(targetDir, 0755)

	if err != nil {
		return fmt.Errorf("failed to create target dir: %w", err)
	}

	for _, f := range r.File {
		fpath := filepath.Join(targetDir, f.Name)

		// ZipSlip check
		if !strings.HasPrefix(fpath, filepath.Clean(targetDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			err = os.MkdirAll(fpath, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}

		err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm)
		if err != nil {
			return err
		}

		var outFile *os.File
		outFile, err = os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		var rc io.ReadCloser
		rc, err = f.Open()
		if err != nil {
			_ = outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		_ = outFile.Close()
		_ = rc.Close()

		if err != nil {
			return err
		}
	}

	log.Info("Successfully downloaded and extracted credits contract.")
	return nil
}
