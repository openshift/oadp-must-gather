package gather

import (
	"bytes"
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/downloadrequest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// isExplicitNamespaceList reports whether included unambiguously names the exact set
// of namespaces to inspect, with no further resolution needed.
//
// It's false whenever the namespace set can only be known by asking what Velero
// actually did: included is empty (defaults to "all namespaces"); included is the
// literal "*"; excluded is non-empty (Velero subtracts it from included, so included
// alone overstates the set); or any entry contains a glob metacharacter (Velero's
// pkg/util/wildcard expands "*", "?", "[...]" patterns against the live namespace list
// at backup/restore time — see wildcard.containsWildcardPattern upstream). In all of
// those cases the caller must fall back to the actual BackupResourceList/
// RestoreResourceList contents instead of trusting the spec field literally.
//
// TODO(velero-io/velero#9772): if namespace selection by ConfigMap-based label
// selector (independent of IncludedNamespaces) lands, a Backup could still leave
// IncludedNamespaces empty while relying on that mechanism — already covered here
// since an empty included falls through to the resourceList fallback.
func isExplicitNamespaceList(included, excluded []string) bool {
	if len(included) == 0 || len(excluded) != 0 {
		return false
	}
	for _, namespace := range included {
		if namespace == "*" || strings.ContainsAny(namespace, "*?[") {
			return false
		}
	}
	return true
}

// namespacesFromResourceList extracts the distinct set of namespaces referenced by a
// Velero Backup/Restore resource list (as returned by the BackupResourceList/
// RestoreResourceList download targets). Each map value is a list of entries shaped
// either "namespace/name" (namespaced resource) or "name" (cluster-scoped resource,
// skipped here since it has no namespace).
func namespacesFromResourceList(resourceList map[string][]string) []string {
	seen := map[string]struct{}{}
	for _, entries := range resourceList {
		for _, entry := range entries {
			namespace, _, found := strings.Cut(entry, "/")
			if !found {
				continue
			}
			seen[namespace] = struct{}{}
		}
	}

	namespaces := make([]string, 0, len(seen))
	for namespace := range seen {
		namespaces = append(namespaces, namespace)
	}
	sort.Strings(namespaces)
	return namespaces
}

// BackupWorkloadNamespaces returns the application namespaces a Backup captures data
// from: backup.Spec.IncludedNamespaces when it unambiguously names them (see
// isExplicitNamespaceList), otherwise the namespaces actually present in the backup,
// discovered via the BackupResourceList download target.
func BackupWorkloadNamespaces(ctx context.Context, clusterClient client.Client, backup velerov1.Backup, timeout time.Duration, skipTLS bool) ([]string, error) {
	included := backup.Spec.IncludedNamespaces
	if isExplicitNamespaceList(included, backup.Spec.ExcludedNamespaces) {
		return included, nil
	}

	writeTo := &bytes.Buffer{}
	if err := downloadrequest.Stream(ctx, clusterClient, backup.Namespace, backup.Name, velerov1.DownloadTargetKindBackupResourceList, writeTo, timeout, skipTLS, ""); err != nil {
		return nil, err
	}

	resourceList := map[string][]string{}
	if err := json.Unmarshal(writeTo.Bytes(), &resourceList); err != nil {
		return nil, err
	}
	return namespacesFromResourceList(resourceList), nil
}

// RestoreWorkloadNamespaces returns the application namespaces a Restore actually
// writes objects into: restore.Spec.IncludedNamespaces (remapped through
// restore.Spec.NamespaceMapping) when it unambiguously names them (see
// isExplicitNamespaceList), otherwise the namespaces actually restored into,
// discovered via the RestoreResourceList download target.
func RestoreWorkloadNamespaces(ctx context.Context, clusterClient client.Client, restore velerov1.Restore, timeout time.Duration, skipTLS bool) ([]string, error) {
	included := restore.Spec.IncludedNamespaces
	if isExplicitNamespaceList(included, restore.Spec.ExcludedNamespaces) {
		namespaces := make([]string, 0, len(included))
		for _, namespace := range included {
			if target, ok := restore.Spec.NamespaceMapping[namespace]; ok {
				namespaces = append(namespaces, target)
			} else {
				namespaces = append(namespaces, namespace)
			}
		}
		return namespaces, nil
	}

	writeTo := &bytes.Buffer{}
	if err := downloadrequest.Stream(ctx, clusterClient, restore.Namespace, restore.Name, velerov1.DownloadTargetKindRestoreResourceList, writeTo, timeout, skipTLS, ""); err != nil {
		return nil, err
	}

	resourceList := map[string][]string{}
	if err := json.Unmarshal(writeTo.Bytes(), &resourceList); err != nil {
		return nil, err
	}
	return namespacesFromResourceList(resourceList), nil
}
