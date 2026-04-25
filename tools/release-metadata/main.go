package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type manifest struct {
	Version     string          `json:"version"`
	Commit      string          `json:"commit"`
	GeneratedAt string          `json:"generated_at"`
	Assets      []manifestAsset `json:"assets"`
}

type manifestAsset struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type goModule struct {
	Path    string `json:"Path"`
	Version string `json:"Version,omitempty"`
	Main    bool   `json:"Main,omitempty"`
	Replace *struct {
		Path    string `json:"Path"`
		Version string `json:"Version,omitempty"`
	} `json:"Replace,omitempty"`
}

type spdxDocument struct {
	SPDXVersion       string        `json:"spdxVersion"`
	DataLicense       string        `json:"dataLicense"`
	SPDXID            string        `json:"SPDXID"`
	Name              string        `json:"name"`
	DocumentNamespace string        `json:"documentNamespace"`
	CreationInfo      creationInfo  `json:"creationInfo"`
	Packages          []spdxPackage `json:"packages"`
}

type creationInfo struct {
	Created  string   `json:"created"`
	Creators []string `json:"creators"`
}

type spdxPackage struct {
	Name                  string `json:"name"`
	SPDXID                string `json:"SPDXID"`
	VersionInfo           string `json:"versionInfo,omitempty"`
	DownloadLocation      string `json:"downloadLocation"`
	FilesAnalyzed         bool   `json:"filesAnalyzed"`
	LicenseConcluded      string `json:"licenseConcluded"`
	LicenseDeclared       string `json:"licenseDeclared"`
	CopyrightText         string `json:"copyrightText"`
	PrimaryPackagePurpose string `json:"primaryPackagePurpose,omitempty"`
}

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "usage: release-metadata <manifest|sbom> <release-dir> <version> <commit>")
		os.Exit(2)
	}

	mode := os.Args[1]
	releaseDir := os.Args[2]
	version := os.Args[3]
	commit := ""
	if len(os.Args) > 4 {
		commit = os.Args[4]
	}

	var (
		output any
		err    error
	)
	switch mode {
	case "manifest":
		output, err = buildManifest(releaseDir, version, commit)
	case "sbom":
		var modules []goModule
		modules, err = listGoModules()
		if err == nil {
			output, err = buildSBOM(version, commit, modules)
		}
	default:
		err = fmt.Errorf("unknown mode %q", mode)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildManifest(releaseDir, version, commit string) (manifest, error) {
	entries, err := os.ReadDir(releaseDir)
	if err != nil {
		return manifest{}, err
	}

	result := manifest{
		Version:     version,
		Commit:      commit,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Assets:      make([]manifestAsset, 0),
	}

	for _, entry := range entries {
		if entry.IsDir() || !isReleaseArchive(entry.Name()) {
			continue
		}
		asset, err := manifestEntry(releaseDir, entry)
		if err != nil {
			return manifest{}, err
		}
		result.Assets = append(result.Assets, asset)
	}
	if len(result.Assets) == 0 {
		for _, entry := range entries {
			if entry.IsDir() || isMetadataFile(entry.Name()) {
				continue
			}
			asset, err := manifestEntry(releaseDir, entry)
			if err != nil {
				return manifest{}, err
			}
			result.Assets = append(result.Assets, asset)
		}
	}
	sort.Slice(result.Assets, func(i, j int) bool {
		return result.Assets[i].Name < result.Assets[j].Name
	})
	if len(result.Assets) == 0 {
		return manifest{}, fmt.Errorf("no release archives found in %s", releaseDir)
	}
	return result, nil
}

func manifestEntry(releaseDir string, entry os.DirEntry) (manifestAsset, error) {
	info, err := entry.Info()
	if err != nil {
		return manifestAsset{}, err
	}
	digest, err := fileSHA256(releaseDir, entry.Name())
	if err != nil {
		return manifestAsset{}, err
	}
	return manifestAsset{
		Name:   entry.Name(),
		Size:   info.Size(),
		SHA256: digest,
	}, nil
}

func isReleaseArchive(name string) bool {
	return strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".zip")
}

func isMetadataFile(name string) bool {
	switch name {
	case "SHA256SUMS", "MANIFEST.json", "SBOM.spdx.json":
		return true
	default:
		return false
	}
}

func fileSHA256(dir, name string) (string, error) {
	file, err := os.OpenInRoot(dir, name)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func listGoModules() ([]goModule, error) {
	cmd := exec.Command("go", "list", "-m", "-json", "all")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(strings.NewReader(string(output)))
	var modules []goModule
	for decoder.More() {
		var module goModule
		if err := decoder.Decode(&module); err != nil {
			return nil, err
		}
		modules = append(modules, module)
	}
	return modules, nil
}

func buildSBOM(version, commit string, modules []goModule) (spdxDocument, error) {
	packages := make([]spdxPackage, 0, len(modules))
	for i, module := range modules {
		path := module.Path
		moduleVersion := module.Version
		if module.Main && moduleVersion == "" {
			moduleVersion = version
		}
		if module.Replace != nil {
			path = module.Replace.Path
			if module.Replace.Version != "" {
				moduleVersion = module.Replace.Version
			}
		}
		packages = append(packages, spdxPackage{
			Name:                  path,
			SPDXID:                fmt.Sprintf("SPDXRef-Package-%d", i+1),
			VersionInfo:           moduleVersion,
			DownloadLocation:      "NOASSERTION",
			FilesAnalyzed:         false,
			LicenseConcluded:      "NOASSERTION",
			LicenseDeclared:       "NOASSERTION",
			CopyrightText:         "NOASSERTION",
			PrimaryPackagePurpose: packagePurpose(module),
		})
	}
	if len(packages) == 0 {
		return spdxDocument{}, fmt.Errorf("no Go modules found")
	}

	namespaceCommit := commit
	if namespaceCommit == "" {
		namespaceCommit = "unknown"
	}
	return spdxDocument{
		SPDXVersion:       "SPDX-2.3",
		DataLicense:       "CC0-1.0",
		SPDXID:            "SPDXRef-DOCUMENT",
		Name:              "morfx-" + version,
		DocumentNamespace: "https://github.com/oxhq/morfx/releases/" + version + "/sbom/" + namespaceCommit,
		CreationInfo: creationInfo{
			Created:  time.Now().UTC().Format(time.RFC3339),
			Creators: []string{"Tool: morfx-release-metadata"},
		},
		Packages: packages,
	}, nil
}

func packagePurpose(module goModule) string {
	if module.Main {
		return "APPLICATION"
	}
	return "LIBRARY"
}
