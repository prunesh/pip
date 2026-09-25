package pip_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/prunesh/pip/filter"
)

// --- Rewrite ---

func TestRewriteInstallAddsProgressBarOff(t *testing.T) {
	got, ok := filter.Rewrite([]string{"install", "torch"})
	if !ok {
		t.Fatal("expected rewrite for install")
	}
	var found bool
	for _, a := range got {
		if strings.HasPrefix(a, "--progress-bar") {
			found = true
		}
	}
	if !found {
		t.Fatalf("--progress-bar flag not injected: %v", got)
	}
}

func TestRewriteDownloadAddsProgressBarOff(t *testing.T) {
	_, ok := filter.Rewrite([]string{"download", "requests"})
	if !ok {
		t.Fatal("expected rewrite for download")
	}
}

func TestRewriteNoRewriteWhenAlreadyQuiet(t *testing.T) {
	_, ok := filter.Rewrite([]string{"install", "-q", "requests"})
	if ok {
		t.Fatal("must not rewrite when -q is present")
	}
}

func TestRewriteNoRewriteWhenProgressBarAlreadySet(t *testing.T) {
	_, ok := filter.Rewrite([]string{"install", "--progress-bar=ascii", "requests"})
	if ok {
		t.Fatal("must not rewrite when --progress-bar already set")
	}
}

func TestRewriteNoRewriteForList(t *testing.T) {
	_, ok := filter.Rewrite([]string{"list"})
	if ok {
		t.Fatal("must not rewrite for list")
	}
}

func TestRewriteNoRewriteForShow(t *testing.T) {
	_, ok := filter.Rewrite([]string{"show", "requests"})
	if ok {
		t.Fatal("must not rewrite for show")
	}
}

func TestRewriteNoArgs(t *testing.T) {
	_, ok := filter.Rewrite(nil)
	if ok {
		t.Fatal("must not rewrite with no args")
	}
}

// --- FilterOutput: install success ---

const installSuccessOutput = `Collecting torch==2.0.0
  Downloading torch-2.0.0-cp311-cp311-manylinux1_x86_64.whl (619.9 MB)
     ━━━━━━━━━━━━━━━━━━━━━━━━━ 619.9/619.9 MB 8.2 MB/s eta 0:00:00
Collecting numpy>=1.21.0 (from torch==2.0.0)
  Using cached numpy-1.26.4-cp311-cp311-manylinux_2_17_x86_64.whl (17.3 MB)
Installing collected packages: numpy, torch
Successfully installed numpy-1.26.4 torch-2.0.0
`

func TestFilterInstallSuccess(t *testing.T) {
	out := filter.FilterOutput([]string{"install", "torch"}, installSuccessOutput, 0)

	if strings.Contains(out, "Collecting") {
		t.Error("filtered output must not contain 'Collecting'")
	}
	if strings.Contains(out, "Downloading") {
		t.Error("filtered output must not contain 'Downloading'")
	}
	if strings.Contains(out, "Using cached") {
		t.Error("filtered output must not contain 'Using cached'")
	}
	if !strings.Contains(out, "Successfully installed") {
		t.Error("filtered output must contain 'Successfully installed'")
	}
	if !strings.Contains(out, "Installing collected packages") {
		t.Error("filtered output must contain 'Installing collected packages'")
	}
}

func TestFilterInstallSuccessIsSmaller(t *testing.T) {
	out := filter.FilterOutput([]string{"install", "torch"}, installSuccessOutput, 0)
	if len(out) >= len(installSuccessOutput) {
		t.Errorf("filtered (%d bytes) must be smaller than original (%d bytes)", len(out), len(installSuccessOutput))
	}
}

// --- FilterOutput: already satisfied ---

func TestFilterInstallAlreadySatisfied(t *testing.T) {
	input := "Requirement already satisfied: requests in /usr/lib/python3 (from pip) (2.31.0)\n" +
		"Requirement already satisfied: urllib3 in /usr/lib/python3 (2.0.7)\n" +
		"Requirement already satisfied: certifi in /usr/lib/python3 (2024.2.2)\n" +
		"Requirement already satisfied: idna in /usr/lib/python3 (3.6)\n" +
		"Requirement already satisfied: charset-normalizer in /usr/lib (3.3.2)\n"

	out := filter.FilterOutput([]string{"install", "requests"}, input, 0)

	// Should keep up to keepSatisfied lines, then show a summary
	lines := nonEmptyLines(out)
	if len(lines) > 5 {
		t.Errorf("too many lines in already-satisfied output: %d\n%s", len(lines), out)
	}
}

// --- FilterOutput: install failure ---

const installErrorOutput = `Collecting nonexistent-package==99.0.0
  Downloading nonexistent-package-99.0.0.tar.gz
ERROR: Could not find a version that satisfies the requirement nonexistent-package==99.0.0 (from versions: none)
ERROR: No matching distribution found for nonexistent-package==99.0.0
`

func TestFilterInstallError(t *testing.T) {
	out := filter.FilterOutput([]string{"install", "nonexistent-package==99.0.0"}, installErrorOutput, 1)

	if !strings.Contains(out, "ERROR") {
		t.Error("filtered error output must contain 'ERROR'")
	}
	if strings.Contains(out, "Downloading") {
		t.Error("filtered error output must not contain 'Downloading'")
	}
}

// --- FilterOutput: uninstall ---

func TestFilterUninstallSuccess(t *testing.T) {
	input := "Found existing installation: requests 2.31.0\nSuccessfully uninstalled requests-2.31.0\n"
	out := filter.FilterOutput([]string{"uninstall", "-y", "requests"}, input, 0)

	if !strings.Contains(out, "Successfully uninstalled") {
		t.Errorf("expected uninstall success line, got: %q", out)
	}
	if strings.Contains(out, "Found existing installation") {
		t.Error("filtered uninstall must not contain verbose preamble")
	}
}

func TestFilterUninstallError(t *testing.T) {
	input := "WARNING: Skipping requests as it is not installed.\n"
	out := filter.FilterOutput([]string{"uninstall", "requests"}, input, 1)
	if out != input {
		t.Errorf("error output must pass through unchanged, got: %q", out)
	}
}

// --- FilterOutput: show ---

const showOutput = `Name: requests
Version: 2.31.0
Summary: Python HTTP for Humans.
Home-page: https://requests.readthedocs.io
Author: Kenneth Reitz
Author-email: me@kennethreitz.org
License: Apache 2.0
Location: /usr/lib/python3/dist-packages
Requires: certifi, charset-normalizer, idna, urllib3
Required-by: pip-api
`

func TestFilterShow(t *testing.T) {
	out := filter.FilterOutput([]string{"show", "requests"}, showOutput, 0)

	for _, want := range []string{"Name:", "Version:", "Location:", "Requires:", "Required-by:"} {
		if !strings.Contains(out, want) {
			t.Errorf("filtered show must contain %q", want)
		}
	}
	for _, unwanted := range []string{"Author:", "License:", "Home-page:", "Summary:"} {
		if strings.Contains(out, unwanted) {
			t.Errorf("filtered show must not contain %q", unwanted)
		}
	}
}

func TestFilterShowIsSmaller(t *testing.T) {
	out := filter.FilterOutput([]string{"show", "requests"}, showOutput, 0)
	if len(out) >= len(showOutput) {
		t.Errorf("filtered show (%d bytes) must be smaller than original (%d bytes)", len(out), len(showOutput))
	}
}

// --- FilterOutput: list ---

func TestFilterListPassthroughWhenSmall(t *testing.T) {
	input := "Package    Version\n---------- -------\nrequests   2.31.0\nnumpy      1.26.4\n"
	out := filter.FilterOutput([]string{"list"}, input, 0)
	if out != input {
		t.Errorf("small list must pass through unchanged, got: %q", out)
	}
}

func TestFilterListTruncatesWhenLarge(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("Package    Version\n---------- -------\n")
	for i := 0; i < 60; i++ {
		sb.WriteString(strings.Repeat("x", 10))
		sb.WriteString("   1.0.0\n")
	}
	input := sb.String()

	out := filter.FilterOutput([]string{"list"}, input, 0)
	if !strings.Contains(out, "more packages") {
		t.Error("large list must show truncation notice")
	}
	if len(out) >= len(input) {
		t.Errorf("large list filter (%d bytes) must be smaller than original (%d bytes)", len(out), len(input))
	}
}

// --- FilterOutput: passthrough subcommands ---

func TestFilterFreezePassthrough(t *testing.T) {
	input := "requests==2.31.0\nnumpy==1.26.4\n"
	out := filter.FilterOutput([]string{"freeze"}, input, 0)
	if out != input {
		t.Errorf("freeze must pass through unchanged")
	}
}

func TestFilterCheckPassthrough(t *testing.T) {
	input := "No broken requirements found.\n"
	out := filter.FilterOutput([]string{"check"}, input, 0)
	if out != input {
		t.Errorf("check must pass through unchanged")
	}
}

// --- ID constant ---

func TestID(t *testing.T) {
	if filter.ID != "prunesh/pip" {
		t.Fatalf("ID %q does not follow author/<cmd> rule", filter.ID)
	}
}

// --- prunesh.json manifest ---

func TestManifest(t *testing.T) {
	data, err := os.ReadFile("prunesh.json")
	if err != nil {
		t.Fatalf("read prunesh.json: %v", err)
	}

	var manifest struct {
		ID                 string   `json:"id"`
		Command            string   `json:"command"`
		Platforms          []string `json:"platforms"`
		Contract           string   `json:"contract"`
		PruneshCoreVersion struct {
			Version    string `json:"version"`
			Constraint string `json:"constraint"`
		} `json:"prunesh-core-version"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse prunesh.json: %v", err)
	}
	if manifest.ID != filter.ID {
		t.Fatalf("manifest id %q != code id %q", manifest.ID, filter.ID)
	}
	if manifest.Command != filter.Command {
		t.Fatalf("manifest command %q != code command %q", manifest.Command, filter.Command)
	}
	if manifest.Contract != "stdin/v1" {
		t.Fatalf("unexpected contract: %q", manifest.Contract)
	}
	if manifest.PruneshCoreVersion.Version == "" {
		t.Fatal("prunesh-core-version.version must not be empty")
	}
	if manifest.PruneshCoreVersion.Constraint != "min" && manifest.PruneshCoreVersion.Constraint != "exact" {
		t.Fatalf("unexpected prunesh-core-version.constraint: %q", manifest.PruneshCoreVersion.Constraint)
	}
	if len(manifest.Platforms) == 0 {
		t.Fatal("platforms must not be empty")
	}
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}
