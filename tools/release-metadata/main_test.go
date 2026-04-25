package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildManifestIncludesAssetsAndSHA256(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "morfx_0.4.0_linux_amd64.tar.gz"), []byte("archive"), 0o600); err != nil {
		t.Fatalf("write archive: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SHA256SUMS"), []byte("hash  asset\n"), 0o600); err != nil {
		t.Fatalf("write checksums: %v", err)
	}

	manifest, err := buildManifest(dir, "v0.4.0", "abc1234")
	if err != nil {
		t.Fatalf("buildManifest returned error: %v", err)
	}

	if manifest.Version != "v0.4.0" || manifest.Commit != "abc1234" {
		t.Fatalf("unexpected manifest identity: %+v", manifest)
	}
	if len(manifest.Assets) != 1 {
		t.Fatalf("expected one release asset, got %+v", manifest.Assets)
	}
	if manifest.Assets[0].Name != "morfx_0.4.0_linux_amd64.tar.gz" {
		t.Fatalf("unexpected asset: %+v", manifest.Assets[0])
	}
	if manifest.Assets[0].SHA256 == "" {
		t.Fatalf("expected sha256 digest in manifest: %+v", manifest.Assets[0])
	}
}

func TestBuildSBOMIncludesGoModules(t *testing.T) {
	sbom, err := buildSBOM("v0.4.0", "abc1234", []goModule{
		{Path: "github.com/oxhq/morfx", Version: "v0.4.0", Main: true},
		{Path: "github.com/smacker/go-tree-sitter", Version: "v0.0.0-20240827094217-dd81d9e9be82"},
	})
	if err != nil {
		t.Fatalf("buildSBOM returned error: %v", err)
	}

	if sbom.SPDXID != "SPDXRef-DOCUMENT" {
		t.Fatalf("unexpected SPDX id: %+v", sbom)
	}
	if len(sbom.Packages) != 2 {
		t.Fatalf("expected two SPDX packages, got %+v", sbom.Packages)
	}
	if sbom.Packages[0].Name != "github.com/oxhq/morfx" {
		t.Fatalf("expected root module first, got %+v", sbom.Packages)
	}
}
