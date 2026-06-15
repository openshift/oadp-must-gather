package templates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"sigs.k8s.io/yaml"
)

func TestReplaceBackupStorageLocationsSectionFromMustGather(t *testing.T) {
	mustGatherBase := os.Getenv("MUST_GATHER_BSL_PATH")
	if mustGatherBase == "" {
		t.Skip("Set MUST_GATHER_BSL_PATH to a must-gather clusters/<id> directory to run this test")
	}

	pattern := filepath.Join(mustGatherBase, "namespaces", "*", "velero.io", "backupstoragelocations", "backupstoragelocations.yaml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob error: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no BSL YAML files found matching %s", pattern)
	}

	allBSLs := &velerov1.BackupStorageLocationList{}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}

		var raw struct {
			Items []json.RawMessage `json:"items"`
		}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			t.Fatalf("unmarshal list from %s: %v", f, err)
		}

		for _, item := range raw.Items {
			var bsl velerov1.BackupStorageLocation
			if err := json.Unmarshal(item, &bsl); err != nil {
				t.Fatalf("unmarshal BSL item from %s: %v", f, err)
			}
			allBSLs.Items = append(allBSLs.Items, bsl)
		}
	}

	t.Logf("Loaded %d BSLs from %d files", len(allBSLs.Items), len(files))
	for _, bsl := range allBSLs.Items {
		s3Url := ""
		if bsl.Spec.Config != nil {
			s3Url = bsl.Spec.Config["s3Url"]
		}
		bucket := ""
		prefix := ""
		if bsl.Spec.ObjectStorage != nil {
			bucket = bsl.Spec.ObjectStorage.Bucket
			prefix = bsl.Spec.ObjectStorage.Prefix
		}
		t.Logf("  %s/%s  provider=%s bucket=%s prefix=%s s3Url=%s",
			bsl.Namespace, bsl.Name, bsl.Spec.Provider, bucket, prefix, s3Url)
	}

	for _, key := range summaryTemplateReplacesKeys {
		summaryTemplateReplaces[key] = ""
	}

	outputPath := t.TempDir() + "/"
	ReplaceBackupStorageLocationsSection(outputPath, allBSLs)

	md := "# BSL Summary (test output)\n\n"
	md += "## Errors\n\n"
	if errs := summaryTemplateReplaces["ERRORS"]; errs != "" {
		md += errs
	} else {
		md += "No errors\n\n"
	}
	md += "## BackupStorageLocations (BSLs)\n\n"
	md += summaryTemplateReplaces["BACKUP_STORAGE_LOCATIONS"]

	outFile := "/tmp/oadp-bsl-summary.md"
	if err := os.WriteFile(outFile, []byte(md), 0644); err != nil {
		t.Fatalf("writing %s: %v", outFile, err)
	}
	t.Logf("Markdown written to %s", outFile)
	fmt.Println(md)
}
