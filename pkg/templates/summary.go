package templates

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	volumesnapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v8/apis/volumesnapshot/v1"
	nac1alpha1 "github.com/migtools/oadp-non-admin/api/v1alpha1"
	vmfrv1alpha1 "github.com/migtools/oadp-vm-file-restore/api/v1alpha1"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	oadpv1alpha1 "github.com/openshift/oadp-operator/api/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	velerov2alpha1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v2alpha1"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/downloadrequest"
	"github.com/vmware-tanzu/velero/pkg/cmd/util/output"
	"github.com/vmware-tanzu/velero/pkg/label"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/oadp-must-gather/pkg/gvk"
)

const (
	FilePermission   = 0644
	FolderPermission = 0777
	IBMCloudConstant = "IBMCloud"
)

var (
	summaryTemplateReplacesKeys = []string{
		"MUST_GATHER_VERSION",
		"ERRORS",
		"CLUSTER_ID", "OCP_VERSION", "CLOUD", "ARCH", "CLUSTER_VERSION",
		"OADP_VERSIONS",
		"DATA_PROTECTION_APPLICATIONS",
		"DATA_PROTECTION_TESTS",
		"CLOUD_STORAGES",
		"BACKUP_STORAGE_LOCATIONS",
		"VOLUME_SNAPSHOT_LOCATIONS",
		"BACKUPS",
		"RESTORES",
		"SCHEDULES",
		"BACKUPS_REPOSITORIES",
		"DATA_UPLOADS",
		"DATA_DOWNLOADS",
		"POD_VOLUME_BACKUPS",
		"POD_VOLUME_RESTORES",
		"DOWNLOAD_REQUESTS",
		"DELETE_BACKUP_REQUESTS",
		"SERVER_STATUS_REQUESTS",
		"NON_ADMIN_BACKUP_STORAGE_LOCATION_REQUESTS",
		"NON_ADMIN_BACKUP_STORAGE_LOCATIONS",
		"NON_ADMIN_BACKUPS",
		"NON_ADMIN_RESTORES",
		"NON_ADMIN_DOWNLOAD_REQUESTS",
		"VIRTUAL_MACHINE_BACKUPS_DISCOVERIES",
		"VIRTUAL_MACHINE_FILE_RESTORES",
		"STORAGE_CLASSES",
		"VOLUME_SNAPSHOT_CLASSES",
		"CSI_DRIVERS",
		"CUSTOM_RESOURCE_DEFINITION",
	}
	summaryTemplateReplaces = map[string]string{}
)

const summaryTemplate = `# OADP must-gather summary version <<MUST_GATHER_VERSION>>

[OADP Frequently Asked Questions](https://access.redhat.com/articles/5456281)

# Table of Contents

- [Errors](#errors)
- [Cluster information](#cluster-information)
- [OADP operator installation information](#oadp-operator-installation-information)
    - [DataProtectionApplications (DPAs)](#dataprotectionapplications-dpas)
	- [DataProtectionTests (DPTs)](#dataprotectiontests-dpts)
    - [CloudStorages](#cloudstorages)
    - [BackupStorageLocations (BSLs)](#backupstoragelocations-bsls)
    - [VolumeSnapshotLocations (VSLs)](#volumesnapshotlocations-vsls)
    - [Backups](#backups)
    - [Restores](#restores)
    - [Schedules](#schedules)
    - [BackupRepositories](#backuprepositories)
    - [DataUploads](#datauploads)
    - [DataDownloads](#datadownloads)
    - [PodVolumeBackups](#podvolumebackups)
    - [PodVolumeRestores](#podvolumerestores)
    - [DownloadRequests](#downloadrequests)
    - [DeleteBackupRequests](#deletebackuprequests)
    - [ServerStatusRequests](#serverstatusrequests)
    - [NonAdminBackupStorageLocationRequests](#nonadminbackupstoragelocationrequests)
    - [NonAdminBackupStorageLocations](#nonadminbackupstoragelocations)
    - [NonAdminBackups](#nonadminbackups)
    - [NonAdminRestores](#nonadminrestores)
    - [NonAdminDownloadRequests](#nonadmindownloadrequests)
    - [VirtualMachineBackupsDiscoveries](#virtualmachinebackupsdiscoveries)
    - [VirtualMachineFileRestores](#virtualmachinefilerestores)
- Storage
    - [Available StorageClasses in cluster](#available-storageclasses-in-cluster)
    - [Available VolumeSnapshotClasses in cluster](#available-volumesnapshotclasses-in-cluster)
    - [Available CSIDrivers in cluster](#available-csidrivers-in-cluster)
- [CustomResourceDefinitions](#customresourcedefinitions)

## Errors

<<ERRORS>>

## Cluster information

| Cluster ID | OpenShift version | Cloud provider | Architecture |
| ---------- | ----------------- | -------------- | ------------ |
| <<CLUSTER_ID>> | <<OCP_VERSION>> | <<CLOUD>> | <<ARCH>> |

<<CLUSTER_VERSION>>

## OADP operator installation information

<<OADP_VERSIONS>>

### DataProtectionApplications (DPAs)

<<DATA_PROTECTION_APPLICATIONS>>

### DataProtectionTests (DPTs)

<<DATA_PROTECTION_TESTS>>

### CloudStorages

<<CLOUD_STORAGES>>

### BackupStorageLocations (BSLs)

<<BACKUP_STORAGE_LOCATIONS>>

### VolumeSnapshotLocations (VSLs)

<<VOLUME_SNAPSHOT_LOCATIONS>>

### Backups

<<BACKUPS>>

### Restores

<<RESTORES>>

### Schedules

<<SCHEDULES>>

### BackupRepositories

<<BACKUPS_REPOSITORIES>>

### DataUploads

<<DATA_UPLOADS>>

### DataDownloads

<<DATA_DOWNLOADS>>

### PodVolumeBackups

<<POD_VOLUME_BACKUPS>>

### PodVolumeRestores

<<POD_VOLUME_RESTORES>>

### DownloadRequests

<<DOWNLOAD_REQUESTS>>

### DeleteBackupRequests

<<DELETE_BACKUP_REQUESTS>>

### ServerStatusRequests

<<SERVER_STATUS_REQUESTS>>

### NonAdminBackupStorageLocationRequests

<<NON_ADMIN_BACKUP_STORAGE_LOCATION_REQUESTS>>

### NonAdminBackupStorageLocations

<<NON_ADMIN_BACKUP_STORAGE_LOCATIONS>>

### NonAdminBackups

<<NON_ADMIN_BACKUPS>>

### NonAdminRestores

<<NON_ADMIN_RESTORES>>

### NonAdminDownloadRequests

<<NON_ADMIN_DOWNLOAD_REQUESTS>>

### VirtualMachineBackupsDiscoveries

<<VIRTUAL_MACHINE_BACKUPS_DISCOVERIES>>

### VirtualMachineFileRestores

<<VIRTUAL_MACHINE_FILE_RESTORES>>

## Available StorageClasses in cluster

<<STORAGE_CLASSES>>

## Available VolumeSnapshotClasses in cluster

<<VOLUME_SNAPSHOT_CLASSES>>

## Available CSIDrivers in cluster

<<CSI_DRIVERS>>

> **Note:** check [supported Container Storage Interface drivers for OpenShift](https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/storage/using-container-storage-interface-csi#csi-drivers-supported_persistent-storage-csi)

## CustomResourceDefinitions

<<CUSTOM_RESOURCE_DEFINITION>>
`

func init() {
	for _, key := range summaryTemplateReplacesKeys {
		summaryTemplateReplaces[key] = ""
	}
}

func ReplaceMustGatherVersion(version string) {
	summaryTemplateReplaces["MUST_GATHER_VERSION"] = "`" + version + "`"
}

func ReplaceClusterInformationSection(
	outputPath string,
	clusterID string,
	clusterVersion *openshiftconfigv1.ClusterVersion,
	infrastructureList *openshiftconfigv1.InfrastructureList,
	nodeList *corev1.NodeList,
) {
	summaryTemplateReplaces["CLUSTER_ID"] = clusterID

	summaryTemplateReplaces["OCP_VERSION"] = clusterVersion.Status.Desired.Version
	summaryTemplateReplaces["CLUSTER_VERSION"] = createYAML(outputPath, "cluster-scoped-resources/config.openshift.io/clusterversions.yaml", clusterVersion)
	cloudProvider := ""

	if infrastructureList != nil && len(infrastructureList.Items) != 0 {
		cloudProvider = string(infrastructureList.Items[0].Spec.PlatformSpec.Type)
		summaryTemplateReplaces["CLOUD"] = cloudProvider
	} else {
		summaryTemplateReplaces["CLOUD"] = "❌ no Infrastructure found in cluster"
		summaryTemplateReplaces["ERRORS"] += "⚠️ No Infrastructure found in cluster\n\n"
	}
	if strings.EqualFold(cloudProvider, IBMCloudConstant) {
		summaryTemplateReplaces["CLOUD"] += " [WARNING:](https://access.redhat.com/articles/5456281#known-issues-with-cloud-providers-and-hyperscalers-18)"
	}

	if nodeList != nil && len(nodeList.Items) != 0 {
		architectureText := ""
		for _, node := range nodeList.Items {
			arch := node.Status.NodeInfo.OperatingSystem + "/" + node.Status.NodeInfo.Architecture
			if len(architectureText) == 0 {
				architectureText += arch
			} else {
				if !strings.Contains(architectureText, arch) {
					architectureText += " | " + arch
				}
			}
		}
		summaryTemplateReplaces["ARCH"] = architectureText
	} else {
		summaryTemplateReplaces["ARCH"] = "❌ no Node found in cluster"
		summaryTemplateReplaces["ERRORS"] += "⚠️ No Node found in cluster\n\n"
	}
}

func ReplaceOADPOperatorInstallationSection(
	outputPath string,
	importantCSVsByNamespace map[string][]operatorsv1alpha1.ClusterServiceVersion,
	importantSubscriptionsByNamespace map[string][]operatorsv1alpha1.Subscription,
	foundOADP bool,
	foundRelatedProducts bool,
	oldOADPError string,
	oadpOperatorsText string,
) {
	if len(importantCSVsByNamespace) == 0 {
		summaryTemplateReplaces["OADP_VERSIONS"] = "❌ No OADP Operator was found installed in the cluster\n\nNo related product was found installed in the cluster"
		summaryTemplateReplaces["ERRORS"] += "🚫 No OADP Operator was found installed in the cluster\n\n"
	} else {
		for namespace, csvs := range importantCSVsByNamespace {
			list := &corev1.List{}
			list.GetObjectKind().SetGroupVersionKind(gvk.ListGVK)
			for _, csv := range csvs {
				csv.GetObjectKind().SetGroupVersionKind(gvk.ClusterServiceVersionGVK)
				list.Items = append(list.Items, runtime.RawExtension{Object: &csv})
			}
			folder := fmt.Sprintf("namespaces/%s/operators.coreos.com/clusterserviceversions", namespace)
			oadpOperatorsText += createYAML(outputPath, folder+"/clusterserviceversions.yaml", list)
		}
		for namespace, subscriptions := range importantSubscriptionsByNamespace {
			list := &corev1.List{}
			list.GetObjectKind().SetGroupVersionKind(gvk.ListGVK)
			for _, subscription := range subscriptions {
				subscription.GetObjectKind().SetGroupVersionKind(gvk.SubscriptionsGVK)
				list.Items = append(list.Items, runtime.RawExtension{Object: &subscription})
			}
			folder := fmt.Sprintf("namespaces/%s/operators.coreos.com/subscriptions", namespace)
			oadpOperatorsText += createYAML(outputPath, folder+"/subscriptions.yaml", list)
		}
		if len(oldOADPError) > 0 {
			summaryTemplateReplaces["ERRORS"] += oldOADPError
		}
		if !foundOADP {
			summaryTemplateReplaces["OADP_VERSIONS"] += "❌ No OADP Operator was found installed in the cluster\n\n"
			summaryTemplateReplaces["ERRORS"] += "🚫 No OADP Operator was found installed in the cluster\n\n"
		}
		summaryTemplateReplaces["OADP_VERSIONS"] += oadpOperatorsText
		if !foundRelatedProducts {
			summaryTemplateReplaces["OADP_VERSIONS"] += "No related product was found installed in the cluster\n\n"
		}
		summaryTemplateReplaces["OADP_VERSIONS"] += fmt.Sprintf("For information about all objects collected in each namespace, check [`%[1]snamespaces`](%[1]snamespaces) folder", outputPath)
	}
}

func ReplaceDataProtectionApplicationsSection(outputPath string, dataProtectionApplicationList *oadpv1alpha1.DataProtectionApplicationList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "DATA_PROTECTION_APPLICATIONS",
		APIGroup:     "oadp.openshift.io",
		ResourceName: "dataprotectionapplications",
		GVK:          gvk.DataProtectionApplicationGVK,
		TableHeader:  "| Namespace | Name | spec.unsupportedOverrides | status.conditions[0] | yaml |\n| --- | --- | --- | --- | --- |\n",
		EmptyMessage: "No DataProtectionApplication was found in the cluster",
		ErrorOnEmpty: true,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			dpa := item.(*oadpv1alpha1.DataProtectionApplication)
			errMsg := ""

			unsupportedOverridesText := "false"
			if dpa.Spec.UnsupportedOverrides != nil {
				for _, value := range dpa.Spec.UnsupportedOverrides {
					if value != "" {
						errMsg += fmt.Sprintf(
							"⚠️ DataProtectionApplication **%v** in **%v** namespace is using **unsupportedOverrides**\n\n",
							dpa.Name, dpa.Namespace,
						)
						unsupportedOverridesText = "⚠️ true"
						break
					}
				}
			}

			dpaStatus := ""
			if len(dpa.Status.Conditions) == 0 {
				dpaStatus = "⚠️ no status"
				errMsg += fmt.Sprintf(
					"⚠️ DataProtectionApplication **%v** with **no status** in **%v** namespace\n\n",
					dpa.Name, dpa.Namespace,
				)
			} else {
				condition := dpa.Status.Conditions[0]
				if condition.Status == v1.ConditionTrue {
					dpaStatus = fmt.Sprintf("✅ status %s: %s", condition.Type, condition.Status)
				} else {
					dpaStatus = fmt.Sprintf("❌ status %s: %s", condition.Type, condition.Status)
					errMsg += fmt.Sprintf(
						"❌ DataProtectionApplication **%v** with **status %s: %s** in **%v** namespace\n\n",
						dpa.Name, condition.Type, condition.Status, dpa.Namespace,
					)
				}
			}

			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			row := fmt.Sprintf(
				"| %v | %v | %v | %v | %s |\n",
				dpa.Namespace, dpa.Name, unsupportedOverridesText, dpaStatus, link,
			)
			return row, errMsg
		},
	}, dataProtectionApplicationList)
}

func ReplaceDataProtectionTestsSection(outputPath string, dptList *oadpv1alpha1.DataProtectionTestList) {
	if dptList != nil && len(dptList.Items) != 0 {
		dataProtectionTestsByNamespace := map[string][]oadpv1alpha1.DataProtectionTest{}

		for _, dpt := range dptList.Items {
			dataProtectionTestsByNamespace[dpt.Namespace] = append(dataProtectionTestsByNamespace[dpt.Namespace], dpt)
		}

		summaryTemplateReplaces["DATA_PROTECTION_TESTS"] = ""
		summaryTemplateReplaces["DATA_PROTECTION_TESTS"] += "| Name | Phase | Last Tested | Upload Speed (MBps) | Encryption | Versioning | Snapshots | Age | YAML |\n"
		summaryTemplateReplaces["DATA_PROTECTION_TESTS"] += "| ---- | ----- | ----------- | ------------------- | ---------- | -----------| --------- | --- | ---- |\n"

		for namespace, dpts := range dataProtectionTestsByNamespace {
			folder := fmt.Sprintf("namespaces/%s/oadp.openshift.io/dataprotectiontests", namespace)
			file := folder + "/dataprotectiontests.yaml"

			list := &corev1.List{}
			list.GetObjectKind().SetGroupVersionKind(gvk.ListGVK)

			for _, dpt := range dpts {
				dpt.GetObjectKind().SetGroupVersionKind(gvk.DataProtectionTestGVK)
				list.Items = append(list.Items, runtime.RawExtension{Object: &dpt})

				// Fields
				name := dpt.Name

				phase := "⚠️ Unknown"
				if dpt.Status.Phase != "" {
					phase = string(dpt.Status.Phase)
				}

				lastTested := "N/A"
				if !dpt.Status.LastTested.IsZero() {
					lastTested = humanizeDurationSince(dpt.Status.LastTested.Time)
				}

				uploadSpeed := "⚠️ N/A"
				if dpt.Status.UploadTest.SpeedMbps > 0 {
					uploadSpeed = fmt.Sprintf("%d", dpt.Status.UploadTest.SpeedMbps)
				}

				encryption := dpt.Status.BucketMetadata.EncryptionAlgorithm
				if encryption == "" {
					encryption = "None"
				}

				versioning := dpt.Status.BucketMetadata.VersioningStatus
				if versioning == "" {
					versioning = "None"
				}

				snapshots := "N/A"
				if dpt.Status.SnapshotSummary != "" {
					snapshots = dpt.Status.SnapshotSummary
				}

				age := humanizeDurationSince(dpt.CreationTimestamp.Time)

				yamlLink := fmt.Sprintf("[`yaml`](%s)", file)

				summaryTemplateReplaces["DATA_PROTECTION_TESTS"] += fmt.Sprintf(
					"| %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
					name, phase, lastTested, uploadSpeed, encryption, versioning, snapshots, age, yamlLink,
				)
			}

			createYAML(outputPath, file, list)
		}
	} else {
		summaryTemplateReplaces["DATA_PROTECTION_TESTS"] = "❌ No DataProtectionTest was found in the cluster"
		summaryTemplateReplaces["ERRORS"] += "⚠️ No DataProtectionTest was found in the cluster\n\n"
	}
}

func humanizeDurationSince(t time.Time) string {
	if t.IsZero() {
		return "N/A"
	}
	return time.Since(t).Round(time.Second).String()
}

func ReplaceCloudStoragesSection(outputPath string, cloudStorageList *oadpv1alpha1.CloudStorageList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "CLOUD_STORAGES",
		APIGroup:     "oadp.openshift.io",
		ResourceName: "cloudstorages",
		GVK:          gvk.CloudStorageGVK,
		TableHeader:  "| Namespace | Name | yaml |\n| --- | --- | --- |\n",
		EmptyMessage: "No CloudStorage was found in the cluster",
		ErrorOnEmpty: false,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			return fmt.Sprintf("| %v | %v | %s |\n", item.GetNamespace(), item.GetName(), link), ""
		},
	}, cloudStorageList)
}

func ReplaceBackupStorageLocationsSection(outputPath string, backupStorageLocationList *velerov1.BackupStorageLocationList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "BACKUP_STORAGE_LOCATIONS",
		APIGroup:     "velero.io",
		ResourceName: "backupstoragelocations",
		GVK:          gvk.BackupStorageLocationGVK,
		TableHeader:  "| Namespace | Name | spec.default | status.phase | yaml |\n| --- | --- | --- | --- | --- |\n",
		EmptyMessage: "No BackupStorageLocation was found in the cluster",
		ErrorOnEmpty: true,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			bsl := item.(*velerov1.BackupStorageLocation)
			errMsg := ""

			bslStatus := ""
			switch bslStatusPhase := bsl.Status.Phase; {
			case len(bslStatusPhase) == 0:
				bslStatus = "⚠️ no status phase"
				errMsg += fmt.Sprintf("⚠️ BackupStorageLocation **%v** with **no status phase** in **%v** namespace\n\n", bsl.Name, bsl.Namespace)
			case bslStatusPhase == velerov1.BackupStorageLocationPhaseAvailable:
				bslStatus = fmt.Sprintf("✅ status phase %s", bslStatusPhase)
			default:
				bslStatus = fmt.Sprintf("❌ status phase %s", bslStatusPhase)
				errMsg += fmt.Sprintf("❌ BackupStorageLocation **%v** with **status phase %s** in **%v** namespace\n\n", bsl.Name, bslStatusPhase, bsl.Namespace)
			}

			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			row := fmt.Sprintf(
				"| %v | %v | %t | %v | %s |\n",
				bsl.Namespace, bsl.Name, bsl.Spec.Default, bslStatus, link,
			)
			return row, errMsg
		},
	}, backupStorageLocationList)
}

func ReplaceVolumeSnapshotLocationsSection(outputPath string, volumeSnapshotLocationList *velerov1.VolumeSnapshotLocationList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "VOLUME_SNAPSHOT_LOCATIONS",
		APIGroup:     "velero.io",
		ResourceName: "volumesnapshotlocations",
		GVK:          gvk.VolumeSnapshotLocationGVK,
		TableHeader:  "| Namespace | Name | yaml |\n| --- | --- | --- |\n",
		EmptyMessage: "No VolumeSnapshotLocation was found in the cluster",
		ErrorOnEmpty: false,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			return fmt.Sprintf("| %v | %v | %s |\n", item.GetNamespace(), item.GetName(), link), ""
		},
	}, volumeSnapshotLocationList)
}

func ReplaceBackupsSection(
	outputPath string,
	backupList *velerov1.BackupList,
	clusterClient client.Client,
	deleteBackupRequestList *velerov1.DeleteBackupRequestList,
	podVolumeBackupList *velerov1.PodVolumeBackupList,
	requestTimeot time.Duration,
	skipTLS bool,
) {
	if backupList != nil && len(backupList.Items) != 0 {
		backupsByNamespace := map[string][]velerov1.Backup{}

		for _, backup := range backupList.Items {
			backupsByNamespace[backup.Namespace] = append(backupsByNamespace[backup.Namespace], backup)
		}

		summaryTemplateReplaces["BACKUPS"] += "| Namespace | Name | status.phase | describe | logs | yaml |\n| --- | --- | --- | --- | --- | ---|\n"
		for namespace, backups := range backupsByNamespace {
			list := &corev1.List{}
			list.GetObjectKind().SetGroupVersionKind(gvk.ListGVK)

			folder := fmt.Sprintf("namespaces/%s/velero.io/backups", namespace)
			file := folder + "/backups.yaml"
			for _, backup := range backups {
				backup.GetObjectKind().SetGroupVersionKind(gvk.BackupGVK)
				list.Items = append(list.Items, runtime.RawExtension{Object: &backup})

				backupStatus := ""
				backupStatusPhase := backup.Status.Phase
				if len(backupStatusPhase) == 0 {
					backupStatus = "⚠️ no status phase"
					summaryTemplateReplaces["ERRORS"] += fmt.Sprintf(
						"⚠️ Backup **%v** with **no status phase** in **%v** namespace\n\n",
						backup.Name, namespace,
					)
				} else {
					failedStates := []velerov1.BackupPhase{
						velerov1.BackupPhaseFailed,
						velerov1.BackupPhasePartiallyFailed,
						velerov1.BackupPhaseFinalizingPartiallyFailed,
						velerov1.BackupPhaseWaitingForPluginOperationsPartiallyFailed,
						velerov1.BackupPhaseFailedValidation,
					}
					if backupStatusPhase == velerov1.BackupPhaseCompleted {
						backupStatus = fmt.Sprintf("✅ status phase %s", backupStatusPhase)
					} else if slices.Contains(failedStates, backupStatusPhase) {
						backupStatus = fmt.Sprintf("❌ status phase %s", backupStatusPhase)
						summaryTemplateReplaces["ERRORS"] += fmt.Sprintf(
							"❌ Backup **%v** with **status phase %s** in **%v** namespace\n\n",
							backup.Name, backupStatusPhase, namespace,
						)
					} else {
						backupStatus = fmt.Sprintf("⚠️ status phase %s", backupStatusPhase)
					}
				}

				var relatedDeleteBackupRequests []velerov1.DeleteBackupRequest
				for _, deleteBackupRequest := range deleteBackupRequestList.Items {
					if deleteBackupRequest.Labels[velerov1.BackupNameLabel] == label.GetValidName(backup.Name) &&
						deleteBackupRequest.Labels[velerov1.BackupUIDLabel] == string(backup.UID) {
						relatedDeleteBackupRequests = append(relatedDeleteBackupRequests, deleteBackupRequest)
					}
				}
				var relatedPodVolumeBackupLists []velerov1.PodVolumeBackup
				for _, podVolumeBackup := range podVolumeBackupList.Items {
					if podVolumeBackup.Labels[velerov1.BackupNameLabel] == label.GetValidName(backup.Name) {
						relatedPodVolumeBackupLists = append(relatedPodVolumeBackupLists, podVolumeBackup)
					}
				}

				// TODO caCertFile?
				describeOutput := func(ctx context.Context) string {
					ctx, cancel := context.WithTimeout(ctx, requestTimeot)
					defer cancel()
					return output.DescribeBackup(ctx, clusterClient, &backup, relatedDeleteBackupRequests, relatedPodVolumeBackupLists, true, skipTLS, "")
				}(context.Background())

				writeTo := &bytes.Buffer{}
				// TODO caCertFile?
				err := downloadrequest.Stream(context.Background(), clusterClient, backup.Namespace, backup.Name, velerov1.DownloadTargetKindBackupLog, writeTo, requestTimeot, skipTLS, "")
				var logs string
				if err != nil {
					fmt.Println(err)
					logs = fmt.Sprintf("❌ %s", err)
				} else {
					logs = createFile(
						outputPath,
						folder+"/"+backup.Name+".log",
						writeTo.String(),
						"logs",
					)
				}
				yamlLink := fmt.Sprintf("[`yaml`](%s)", file)
				summaryTemplateReplaces["BACKUPS"] += fmt.Sprintf(
					"| %v | %v | %s | %s | %s | %s |\n",
					namespace, backup.Name,
					backupStatus,
					createFile(
						outputPath,
						folder+"/describe-"+backup.Name+".txt",
						describeOutput,
						"describe",
					),
					logs,
					yamlLink,
				)
			}

			createYAML(outputPath, file, list)
		}
	} else {
		summaryTemplateReplaces["BACKUPS"] = "❌ No Backup was found in the cluster"
	}
}

func ReplaceRestoresSection(
	outputPath string,
	restoreListList *velerov1.RestoreList,
	clusterClient client.Client,
	podVolumeRestoreList *velerov1.PodVolumeRestoreList,
	requestTimeot time.Duration,
	skipTLS bool,
) {
	if restoreListList != nil && len(restoreListList.Items) != 0 {
		restoresByNamespace := map[string][]velerov1.Restore{}

		for _, restore := range restoreListList.Items {
			restoresByNamespace[restore.Namespace] = append(restoresByNamespace[restore.Namespace], restore)
		}

		summaryTemplateReplaces["RESTORES"] += "| Namespace | Name | status.phase | describe | logs | yaml |\n| --- | --- | --- | --- | --- | --- |\n"
		for namespace, restores := range restoresByNamespace {
			list := &corev1.List{}
			list.GetObjectKind().SetGroupVersionKind(gvk.ListGVK)

			folder := fmt.Sprintf("namespaces/%s/velero.io/restores", namespace)
			file := folder + "/restores.yaml"
			for _, restore := range restores {
				restore.GetObjectKind().SetGroupVersionKind(gvk.RestoreGVK)
				list.Items = append(list.Items, runtime.RawExtension{Object: &restore})

				restoreStatus := ""
				restoreStatusPhase := restore.Status.Phase
				if len(restoreStatusPhase) == 0 {
					restoreStatus = "⚠️ no status phase"
					summaryTemplateReplaces["ERRORS"] += fmt.Sprintf(
						"⚠️ Restore **%v** with **no status phase** in **%v** namespace\n\n",
						restore.Name, namespace,
					)
				} else {
					failedStates := []velerov1.RestorePhase{
						velerov1.RestorePhaseFailed,
						velerov1.RestorePhasePartiallyFailed,
						velerov1.RestorePhaseFinalizingPartiallyFailed,
						velerov1.RestorePhaseWaitingForPluginOperationsPartiallyFailed,
						velerov1.RestorePhaseFailedValidation,
					}
					if restoreStatusPhase == velerov1.RestorePhaseCompleted {
						restoreStatus = fmt.Sprintf("✅ status phase %s", restoreStatusPhase)
					} else if slices.Contains(failedStates, restoreStatusPhase) {
						restoreStatus = fmt.Sprintf("❌ status phase %s", restoreStatusPhase)
						summaryTemplateReplaces["ERRORS"] += fmt.Sprintf(
							"❌ Restore **%v** with **status phase %s** in **%v** namespace\n\n",
							restore.Name, restoreStatusPhase, namespace,
						)
					} else {
						restoreStatus = fmt.Sprintf("⚠️ status phase %s", restoreStatusPhase)
					}
				}

				var relatedPodVolumeRestoreLists []velerov1.PodVolumeRestore
				for _, podVolumeRestore := range podVolumeRestoreList.Items {
					if podVolumeRestore.Labels[velerov1.RestoreNameLabel] == label.GetValidName(restore.Name) {
						relatedPodVolumeRestoreLists = append(relatedPodVolumeRestoreLists, podVolumeRestore)
					}
				}

				// TODO caCertFile?
				describeOutput := func(ctx context.Context) string {
					ctx, cancel := context.WithTimeout(ctx, requestTimeot)
					defer cancel()
					return output.DescribeRestore(ctx, clusterClient, &restore, relatedPodVolumeRestoreLists, true, skipTLS, "")
				}(context.Background())

				writeTo := &bytes.Buffer{}
				// TODO caCertFile?
				err := downloadrequest.Stream(context.Background(), clusterClient, restore.Namespace, restore.Name, velerov1.DownloadTargetKindRestoreLog, writeTo, requestTimeot, skipTLS, "")
				var logs string
				if err != nil {
					fmt.Println(err)
					logs = fmt.Sprintf("❌ %s", err)
				} else {
					logs = createFile(
						outputPath,
						folder+"/"+restore.Name+".log",
						writeTo.String(),
						"logs",
					)
				}

				yamllink := fmt.Sprintf("[`yaml`](%s)", file)
				summaryTemplateReplaces["RESTORES"] += fmt.Sprintf(
					"| %v | %v | %s | %s | %s | %s |\n",
					namespace, restore.Name,
					restoreStatus,
					createFile(
						outputPath,
						folder+"/describe-"+restore.Name+".txt",
						describeOutput,
						"describe",
					),
					logs,
					yamllink,
				)
			}

			createYAML(outputPath, file, list)
		}
	} else {
		summaryTemplateReplaces["RESTORES"] = "❌ No Restore was found in the cluster"
	}
}

func ReplaceSchedulesSection(outputPath string, scheduleList *velerov1.ScheduleList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "SCHEDULES",
		APIGroup:     "velero.io",
		ResourceName: "schedules",
		GVK:          gvk.ScheduleGVK,
		TableHeader:  "| Namespace | Name | status.phase | yaml |\n| --- | --- | --- | --- |\n",
		EmptyMessage: "No Schedule was found in the cluster",
		ErrorOnEmpty: false,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			schedule := item.(*velerov1.Schedule)
			errMsg := ""

			scheduleStatus := ""
			switch scheduleStatusPhase := schedule.Status.Phase; {
			case len(scheduleStatusPhase) == 0:
				scheduleStatus = "⚠️ no status phase"
				errMsg += fmt.Sprintf("⚠️ Schedule **%v** with **no status phase** in **%v** namespace\n\n", schedule.Name, schedule.Namespace)
			case scheduleStatusPhase == velerov1.SchedulePhaseEnabled:
				scheduleStatus = fmt.Sprintf("✅ status phase %s", scheduleStatusPhase)
			case scheduleStatusPhase == velerov1.SchedulePhaseFailedValidation:
				scheduleStatus = fmt.Sprintf("❌ status phase %s", scheduleStatusPhase)
				errMsg += fmt.Sprintf("❌ Schedule **%v** with **status phase %s** in **%v** namespace\n\n", schedule.Name, scheduleStatusPhase, schedule.Namespace)
			default:
				scheduleStatus = fmt.Sprintf("⚠️ status phase %s", scheduleStatusPhase)
			}

			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			row := fmt.Sprintf("| %v | %v | %s | %s |\n", schedule.Namespace, schedule.Name, scheduleStatus, link)
			return row, errMsg
		},
	}, scheduleList)
}

func ReplaceBackupRepositoriesSection(outputPath string, backupRepositoryList *velerov1.BackupRepositoryList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "BACKUPS_REPOSITORIES",
		APIGroup:     "velero.io",
		ResourceName: "backuprepositories",
		GVK:          gvk.BackupRepositoryGVK,
		TableHeader:  "| Namespace | Name | status.phase | yaml |\n| --- | --- | --- | --- |\n",
		EmptyMessage: "No BackupRepository was found in the cluster",
		ErrorOnEmpty: false,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			br := item.(*velerov1.BackupRepository)
			errMsg := ""

			status := ""
			switch phase := br.Status.Phase; {
			case len(phase) == 0:
				status = "⚠️ no status phase"
				errMsg += fmt.Sprintf("⚠️ BackupRepository **%v** with **no status phase** in **%v** namespace\n\n", br.Name, br.Namespace)
			case phase == velerov1.BackupRepositoryPhaseReady:
				status = fmt.Sprintf("✅ status phase %s", phase)
			case phase == velerov1.BackupRepositoryPhaseNotReady:
				status = fmt.Sprintf("❌ status phase %s", phase)
				errMsg += fmt.Sprintf("❌ BackupRepository **%v** with **status phase %s** in **%v** namespace\n\n", br.Name, phase, br.Namespace)
			default:
				status = fmt.Sprintf("⚠️ status phase %s", phase)
			}

			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			row := fmt.Sprintf("| %v | %v | %s | %s |\n", br.Namespace, br.Name, status, link)
			return row, errMsg
		},
	}, backupRepositoryList)
}

func ReplaceDataUploadsSection(outputPath string, dataUploadList *velerov2alpha1.DataUploadList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "DATA_UPLOADS",
		APIGroup:     "velero.io",
		ResourceName: "datauploads",
		GVK:          gvk.DataUploadGVK,
		TableHeader:  "| Namespace | Name | status.phase | yaml |\n| --- | --- | --- | --- |\n",
		EmptyMessage: "No DataUpload was found in the cluster",
		ErrorOnEmpty: false,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			du := item.(*velerov2alpha1.DataUpload)
			errMsg := ""

			status := ""
			switch phase := du.Status.Phase; {
			case len(phase) == 0:
				status = "⚠️ no status phase"
				errMsg += fmt.Sprintf("⚠️ DataUpload **%v** with **no status phase** in **%v** namespace\n\n", du.Name, du.Namespace)
			case phase == velerov2alpha1.DataUploadPhaseCompleted:
				status = fmt.Sprintf("✅ status phase %s", phase)
			case slices.Contains([]velerov2alpha1.DataUploadPhase{
				velerov2alpha1.DataUploadPhaseCanceling,
				velerov2alpha1.DataUploadPhaseCanceled,
				velerov2alpha1.DataUploadPhaseFailed,
			}, phase):
				status = fmt.Sprintf("❌ status phase %s", phase)
				errMsg += fmt.Sprintf("❌ DataUpload **%v** with **status phase %s** in **%v** namespace\n\n", du.Name, phase, du.Namespace)
			default:
				status = fmt.Sprintf("⚠️ status phase %s", phase)
			}

			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			row := fmt.Sprintf("| %v | %v | %s | %s |\n", du.Namespace, du.Name, status, link)
			return row, errMsg
		},
	}, dataUploadList)
}

func ReplaceDataDownloadsSection(outputPath string, dataDownloadList *velerov2alpha1.DataDownloadList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "DATA_DOWNLOADS",
		APIGroup:     "velero.io",
		ResourceName: "datadownloads",
		GVK:          gvk.DataDownloadGVK,
		TableHeader:  "| Namespace | Name | status.phase | yaml |\n| --- | --- | --- | --- |\n",
		EmptyMessage: "No DataDownload was found in the cluster",
		ErrorOnEmpty: false,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			dd := item.(*velerov2alpha1.DataDownload)
			errMsg := ""

			status := ""
			switch phase := dd.Status.Phase; {
			case len(phase) == 0:
				status = "⚠️ no status phase"
				errMsg += fmt.Sprintf("⚠️ DataDownload **%v** with **no status phase** in **%v** namespace\n\n", dd.Name, dd.Namespace)
			case phase == velerov2alpha1.DataDownloadPhaseCompleted:
				status = fmt.Sprintf("✅ status phase %s", phase)
			case slices.Contains([]velerov2alpha1.DataDownloadPhase{
				velerov2alpha1.DataDownloadPhaseCanceling,
				velerov2alpha1.DataDownloadPhaseCanceled,
				velerov2alpha1.DataDownloadPhaseFailed,
			}, phase):
				status = fmt.Sprintf("❌ status phase %s", phase)
				errMsg += fmt.Sprintf("❌ DataDownload **%v** with **status phase %s** in **%v** namespace\n\n", dd.Name, phase, dd.Namespace)
			default:
				status = fmt.Sprintf("⚠️ status phase %s", phase)
			}

			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			row := fmt.Sprintf("| %v | %v | %s | %s |\n", dd.Namespace, dd.Name, status, link)
			return row, errMsg
		},
	}, dataDownloadList)
}

func ReplacePodVolumeBackupsSection(outputPath string, podVolumeBackupList *velerov1.PodVolumeBackupList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "POD_VOLUME_BACKUPS",
		APIGroup:     "velero.io",
		ResourceName: "podvolumebackups",
		GVK:          gvk.PodVolumeBackupGVK,
		TableHeader:  "| Namespace | Name | status.phase | yaml |\n| --- | --- | --- | --- |\n",
		EmptyMessage: "No PodVolumeBackup was found in the cluster",
		ErrorOnEmpty: false,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			pvb := item.(*velerov1.PodVolumeBackup)
			errMsg := ""

			status := ""
			switch phase := pvb.Status.Phase; {
			case len(phase) == 0:
				status = "⚠️ no status phase"
				errMsg += fmt.Sprintf("⚠️ PodVolumeBackup **%v** with **no status phase** in **%v** namespace\n\n", pvb.Name, pvb.Namespace)
			case phase == velerov1.PodVolumeBackupPhaseCompleted:
				status = fmt.Sprintf("✅ status phase %s", phase)
			case phase == velerov1.PodVolumeBackupPhaseFailed:
				status = fmt.Sprintf("❌ status phase %s", phase)
				errMsg += fmt.Sprintf("❌ PodVolumeBackup **%v** with **status phase %s** in **%v** namespace\n\n", pvb.Name, phase, pvb.Namespace)
			default:
				status = fmt.Sprintf("⚠️ status phase %s", phase)
			}

			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			row := fmt.Sprintf("| %v | %v | %s | %s |\n", pvb.Namespace, pvb.Name, status, link)
			return row, errMsg
		},
	}, podVolumeBackupList)
}

func ReplacePodVolumeRestoresSection(outputPath string, podVolumeRestoreList *velerov1.PodVolumeRestoreList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "POD_VOLUME_RESTORES",
		APIGroup:     "velero.io",
		ResourceName: "podvolumerestores",
		GVK:          gvk.PodVolumeRestoreGVK,
		TableHeader:  "| Namespace | Name | status.phase | yaml |\n| --- | --- | --- | --- |\n",
		EmptyMessage: "No PodVolumeRestore was found in the cluster",
		ErrorOnEmpty: false,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			pvr := item.(*velerov1.PodVolumeRestore)
			errMsg := ""

			status := ""
			switch phase := pvr.Status.Phase; {
			case len(phase) == 0:
				status = "⚠️ no status phase"
				errMsg += fmt.Sprintf("⚠️ PodVolumeRestore **%v** with **no status phase** in **%v** namespace\n\n", pvr.Name, pvr.Namespace)
			case phase == velerov1.PodVolumeRestorePhaseCompleted:
				status = fmt.Sprintf("✅ status phase %s", phase)
			case phase == velerov1.PodVolumeRestorePhaseFailed:
				status = fmt.Sprintf("❌ status phase %s", phase)
				errMsg += fmt.Sprintf("❌ PodVolumeRestore **%v** with **status phase %s** in **%v** namespace\n\n", pvr.Name, phase, pvr.Namespace)
			default:
				status = fmt.Sprintf("⚠️ status phase %s", phase)
			}

			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			row := fmt.Sprintf("| %v | %v | %s | %s |\n", pvr.Namespace, pvr.Name, status, link)
			return row, errMsg
		},
	}, podVolumeRestoreList)
}

func ReplaceDownloadRequestsSection(outputPath string, downloadRequestList *velerov1.DownloadRequestList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "DOWNLOAD_REQUESTS",
		APIGroup:     "velero.io",
		ResourceName: "downloadrequests",
		GVK:          gvk.DownloadRequestGVK,
		TableHeader:  "| Namespace | Name | status.phase | yaml |\n| --- | --- | --- | --- |\n",
		EmptyMessage: "No DownloadRequest was found in the cluster",
		ErrorOnEmpty: false,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			dr := item.(*velerov1.DownloadRequest)
			errMsg := ""

			status := ""
			switch phase := dr.Status.Phase; {
			case len(phase) == 0:
				status = "⚠️ no status"
				errMsg += fmt.Sprintf("⚠️ DownloadRequest **%v** with **no status** in **%v** namespace\n\n", dr.Name, dr.Namespace)
			case phase == velerov1.DownloadRequestPhaseProcessed:
				status = fmt.Sprintf("✅ status phase %s", phase)
			default:
				status = fmt.Sprintf("⚠️ status phase %s", phase)
			}

			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			row := fmt.Sprintf("| %v | %v | %s | %s |\n", dr.Namespace, dr.Name, status, link)
			return row, errMsg
		},
	}, downloadRequestList)
}

func ReplaceDeleteBackupRequestsSection(outputPath string, deleteBackupRequestList *velerov1.DeleteBackupRequestList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "DELETE_BACKUP_REQUESTS",
		APIGroup:     "velero.io",
		ResourceName: "deletebackuprequests",
		GVK:          gvk.DeleteBackupRequestGVK,
		TableHeader:  "| Namespace | Name | status.phase | yaml |\n| --- | --- | --- | --- |\n",
		EmptyMessage: "No DeleteBackupRequest was found in the cluster",
		ErrorOnEmpty: false,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			dbr := item.(*velerov1.DeleteBackupRequest)
			errMsg := ""

			status := ""
			switch phase := dbr.Status.Phase; {
			case len(phase) == 0:
				status = "⚠️ no status"
				errMsg += fmt.Sprintf("⚠️ DeleteBackupRequest **%v** with **no status** in **%v** namespace\n\n", dbr.Name, dbr.Namespace)
			case phase == velerov1.DeleteBackupRequestPhaseProcessed:
				status = fmt.Sprintf("✅ status phase %s", phase)
			default:
				status = fmt.Sprintf("⚠️ status phase %s", phase)
			}

			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			row := fmt.Sprintf("| %v | %v | %s | %s |\n", dbr.Namespace, dbr.Name, status, link)
			return row, errMsg
		},
	}, deleteBackupRequestList)
}

func ReplaceServerStatusRequestsSection(outputPath string, serverStatusRequestList *velerov1.ServerStatusRequestList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "SERVER_STATUS_REQUESTS",
		APIGroup:     "velero.io",
		ResourceName: "serverstatusrequests",
		GVK:          gvk.ServerStatusRequestGVK,
		TableHeader:  "| Namespace | Name | status.phase | yaml |\n| --- | --- | --- | --- |\n",
		EmptyMessage: "No ServerStatusRequest was found in the cluster",
		ErrorOnEmpty: false,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			ssr := item.(*velerov1.ServerStatusRequest)
			errMsg := ""

			status := ""
			switch phase := ssr.Status.Phase; {
			case len(phase) == 0:
				status = "⚠️ no status"
				errMsg += fmt.Sprintf("⚠️ ServerStatusRequest **%v** with **no status** in **%v** namespace\n\n", ssr.Name, ssr.Namespace)
			case phase == velerov1.ServerStatusRequestPhaseProcessed:
				status = fmt.Sprintf("✅ status phase %s", phase)
			default:
				status = fmt.Sprintf("⚠️ status phase %s", phase)
			}

			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			row := fmt.Sprintf("| %v | %v | %s | %s |\n", ssr.Namespace, ssr.Name, status, link)
			return row, errMsg
		},
	}, serverStatusRequestList)
}

func ReplaceNonAdminBackupStorageLocationRequestsSection(outputPath string, nonAdminBackupStorageLocationRequestList *nac1alpha1.NonAdminBackupStorageLocationRequestList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "NON_ADMIN_BACKUP_STORAGE_LOCATION_REQUESTS",
		APIGroup:     "oadp.openshift.io",
		ResourceName: "nonadminbackupstoragelocationrequests",
		GVK:          gvk.NonAdminBackupStorageLocationRequestGVK,
		TableHeader:  "| Namespace | Name | status.phase | yaml |\n| --- | --- | --- | --- |\n",
		EmptyMessage: "No NonAdminBackupStorageLocationRequest was found in the cluster",
		ErrorOnEmpty: false,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			req := item.(*nac1alpha1.NonAdminBackupStorageLocationRequest)
			errMsg := ""

			status := ""
			switch phase := req.Status.Phase; {
			case len(phase) == 0:
				status = "⚠️ no status phase"
				errMsg += fmt.Sprintf("⚠️ NonAdminBackupStorageLocationRequest **%v** with **no status phase** in **%v** namespace\n\n", req.Name, req.Namespace)
			case phase == nac1alpha1.NonAdminBSLRequestPhaseApproved:
				status = fmt.Sprintf("✅ status phase %s", phase)
			case phase == nac1alpha1.NonAdminBSLRequestPhaseRejected:
				status = fmt.Sprintf("❌ status phase %s", phase)
				errMsg += fmt.Sprintf("❌ NonAdminBackupStorageLocationRequest **%v** with **status phase %s** in **%v** namespace\n\n", req.Name, phase, req.Namespace)
			default:
				status = fmt.Sprintf("⚠️ status phase %s", phase)
			}

			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			row := fmt.Sprintf("| %v | %v | %v | %s |\n", req.Namespace, req.Name, status, link)
			return row, errMsg
		},
	}, nonAdminBackupStorageLocationRequestList)
}

func ReplaceNonAdminBackupStorageLocationsSection(outputPath string, nonAdminBackupStorageLocationList *nac1alpha1.NonAdminBackupStorageLocationList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "NON_ADMIN_BACKUP_STORAGE_LOCATIONS",
		APIGroup:     "oadp.openshift.io",
		ResourceName: "nonadminbackupstoragelocations",
		GVK:          gvk.NonAdminBackupStorageLocationGVK,
		TableHeader:  "| Namespace | Name | Approved | status.phase | yaml |\n| --- | --- | --- | --- | --- |\n",
		EmptyMessage: "No NonAdminBackupStorageLocation was found in the cluster",
		ErrorOnEmpty: false,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			bsl := item.(*nac1alpha1.NonAdminBackupStorageLocation)
			errMsg := ""

			bslStatus := ""
			switch bslStatusPhase := bsl.Status.Phase; {
			case len(bslStatusPhase) == 0:
				bslStatus = "⚠️ no status phase"
				errMsg += fmt.Sprintf("⚠️ NonAdminBackupStorageLocation **%v** with **no status phase** in **%v** namespace\n\n", bsl.Name, bsl.Namespace)
			case bslStatusPhase == nac1alpha1.NonAdminPhaseCreated:
				bslStatus = fmt.Sprintf("✅ status phase %s", bslStatusPhase)
			case bslStatusPhase == nac1alpha1.NonAdminPhaseBackingOff:
				bslStatus = fmt.Sprintf("❌ status phase %s", bslStatusPhase)
				errMsg += fmt.Sprintf("❌ NonAdminBackupStorageLocation **%v** with **status phase %s** in **%v** namespace\n\n", bsl.Name, bslStatusPhase, bsl.Namespace)
			default:
				bslStatus = fmt.Sprintf("⚠️ status phase %s", bslStatusPhase)
			}

			bslStatusApproved := ""
			conditionInNABSL := meta.FindStatusCondition(bsl.Status.Conditions, string(nac1alpha1.NonAdminBSLConditionApproved))
			if conditionInNABSL == nil {
				bslStatusApproved = "⚠️ no status condition approved"
				errMsg += fmt.Sprintf("⚠️ NonAdminBackupStorageLocation **%v** with **no status condition approved** in **%v** namespace\n\n", bsl.Name, bsl.Namespace)
			} else {
				if conditionInNABSL.Status == v1.ConditionTrue {
					bslStatusApproved = fmt.Sprintf("✅ status condition approved %s", conditionInNABSL.Status)
				} else {
					bslStatusApproved = fmt.Sprintf("❌ status condition approved %s", conditionInNABSL.Status)
				}
			}

			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			row := fmt.Sprintf("| %v | %v | %s | %v | %s |\n", bsl.Namespace, bsl.Name, bslStatusApproved, bslStatus, link)
			return row, errMsg
		},
	}, nonAdminBackupStorageLocationList)
}

func ReplaceNonAdminBackupsSection(outputPath string, nonAdminBackupList *nac1alpha1.NonAdminBackupList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "NON_ADMIN_BACKUPS",
		APIGroup:     "oadp.openshift.io",
		ResourceName: "nonadminbackups",
		GVK:          gvk.NonAdminBackupGVK,
		TableHeader:  "| Namespace | Name | status.phase | yaml |\n| --- | --- | --- | --- |\n",
		EmptyMessage: "No NonAdminBackup was found in the cluster",
		ErrorOnEmpty: false,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			backup := item.(*nac1alpha1.NonAdminBackup)
			errMsg := ""

			status := ""
			switch phase := backup.Status.Phase; {
			case len(phase) == 0:
				status = "⚠️ no status phase"
				errMsg += fmt.Sprintf("⚠️ NonAdminBackup **%v** with **no status phase** in **%v** namespace\n\n", backup.Name, backup.Namespace)
			case phase == nac1alpha1.NonAdminPhaseCreated:
				status = fmt.Sprintf("✅ status phase %s", phase)
			case phase == nac1alpha1.NonAdminPhaseBackingOff:
				status = fmt.Sprintf("❌ status phase %s", phase)
				errMsg += fmt.Sprintf("❌ NonAdminBackup **%v** with **status phase %s** in **%v** namespace\n\n", backup.Name, phase, backup.Namespace)
			default:
				status = fmt.Sprintf("⚠️ status phase %s", phase)
			}

			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			row := fmt.Sprintf("| %v | %v | %s | %s |\n", backup.Namespace, backup.Name, status, link)
			return row, errMsg
		},
	}, nonAdminBackupList)
}

func ReplaceNonAdminRestoresSection(outputPath string, nonAdminRestoreList *nac1alpha1.NonAdminRestoreList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "NON_ADMIN_RESTORES",
		APIGroup:     "oadp.openshift.io",
		ResourceName: "nonadminrestores",
		GVK:          gvk.NonAdminRestoreGVK,
		TableHeader:  "| Namespace | Name | status.phase | yaml |\n| --- | --- | --- | --- |\n",
		EmptyMessage: "No NonAdminRestore was found in the cluster",
		ErrorOnEmpty: false,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			restore := item.(*nac1alpha1.NonAdminRestore)
			errMsg := ""

			status := ""
			switch phase := restore.Status.Phase; {
			case len(phase) == 0:
				status = "⚠️ no status phase"
				errMsg += fmt.Sprintf("⚠️ NonAdminRestore **%v** with **no status phase** in **%v** namespace\n\n", restore.Name, restore.Namespace)
			case phase == nac1alpha1.NonAdminPhaseCreated:
				status = fmt.Sprintf("✅ status phase %s", phase)
			case phase == nac1alpha1.NonAdminPhaseBackingOff:
				status = fmt.Sprintf("❌ status phase %s", phase)
				errMsg += fmt.Sprintf("❌ NonAdminRestore **%v** with **status phase %s** in **%v** namespace\n\n", restore.Name, phase, restore.Namespace)
			default:
				status = fmt.Sprintf("⚠️ status phase %s", phase)
			}

			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			row := fmt.Sprintf("| %v | %v | %s | %s |\n", restore.Namespace, restore.Name, status, link)
			return row, errMsg
		},
	}, nonAdminRestoreList)
}

func ReplaceNonAdminDownloadRequestsSection(outputPath string, nonAdminDownloadRequestList *nac1alpha1.NonAdminDownloadRequestList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "NON_ADMIN_DOWNLOAD_REQUESTS",
		APIGroup:     "oadp.openshift.io",
		ResourceName: "nonadmindownloadrequests",
		GVK:          gvk.NonAdminDownloadRequestGVK,
		TableHeader:  "| Namespace | Name | status.phase | yaml |\n| --- | --- | --- | --- |\n",
		EmptyMessage: "No NonAdminDownloadRequest was found in the cluster",
		ErrorOnEmpty: false,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			dr := item.(*nac1alpha1.NonAdminDownloadRequest)
			errMsg := ""

			status := ""
			switch phase := dr.Status.Phase; {
			case len(phase) == 0:
				status = "⚠️ no status"
				errMsg += fmt.Sprintf("⚠️ NonAdminDownloadRequest **%v** with **no status** in **%v** namespace\n\n", dr.Name, dr.Namespace)
			case phase == nac1alpha1.NonAdminPhaseCreated:
				status = fmt.Sprintf("✅ status phase %s", phase)
			case phase == nac1alpha1.NonAdminPhaseBackingOff:
				status = fmt.Sprintf("❌ status phase %s", phase)
				errMsg += fmt.Sprintf("❌ NonAdminDownloadRequest **%v** with **status phase %s** in **%v** namespace\n\n", dr.Name, phase, dr.Namespace)
			default:
				status = fmt.Sprintf("⚠️ status phase %s", phase)
			}

			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			row := fmt.Sprintf("| %v | %v | %s | %s |\n", dr.Namespace, dr.Name, status, link)
			return row, errMsg
		},
	}, nonAdminDownloadRequestList)
}

func ReplaceVirtualMachineBackupsDiscoveriesSection(outputPath string, vmBackupsDiscoveryList *vmfrv1alpha1.VirtualMachineBackupsDiscoveryList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "VIRTUAL_MACHINE_BACKUPS_DISCOVERIES",
		APIGroup:     "oadp.openshift.io",
		ResourceName: "virtualmachinebackupsdiscoveries",
		GVK:          gvk.VirtualMachineBackupsDiscoveryGVK,
		TableHeader:  "| Namespace | Name | status.phase | yaml |\n| --- | --- | --- | --- |\n",
		EmptyMessage: "No VirtualMachineBackupsDiscovery was found in the cluster",
		ErrorOnEmpty: false,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			discovery := item.(*vmfrv1alpha1.VirtualMachineBackupsDiscovery)
			errMsg := ""

			status := ""
			switch phase := discovery.Status.Phase; {
			case len(phase) == 0:
				status = "⚠️ no status phase"
				errMsg += fmt.Sprintf("⚠️ VirtualMachineBackupsDiscovery **%v** with **no status phase** in **%v** namespace\n\n", discovery.Name, discovery.Namespace)
			case phase == vmfrv1alpha1.VirtualMachineBackupsDiscoveryPhaseCompleted:
				status = fmt.Sprintf("✅ status phase %s", phase)
			case slices.Contains([]vmfrv1alpha1.VirtualMachineBackupsDiscoveryPhase{
				vmfrv1alpha1.VirtualMachineBackupsDiscoveryPhaseFailed,
				vmfrv1alpha1.VirtualMachineBackupsDiscoveryPhasePartiallyFailed,
			}, phase):
				status = fmt.Sprintf("❌ status phase %s", phase)
				errMsg += fmt.Sprintf("❌ VirtualMachineBackupsDiscovery **%v** with **status phase %s** in **%v** namespace\n\n", discovery.Name, phase, discovery.Namespace)
			default:
				status = fmt.Sprintf("⚠️ status phase %s", phase)
			}

			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			row := fmt.Sprintf("| %v | %v | %s | %s |\n", discovery.Namespace, discovery.Name, status, link)
			return row, errMsg
		},
	}, vmBackupsDiscoveryList)
}

func ReplaceVirtualMachineFileRestoresSection(outputPath string, vmFileRestoreList *vmfrv1alpha1.VirtualMachineFileRestoreList) {
	ReplaceSection(outputPath, SectionConfig{
		TemplateKey:  "VIRTUAL_MACHINE_FILE_RESTORES",
		APIGroup:     "oadp.openshift.io",
		ResourceName: "virtualmachinefilerestores",
		GVK:          gvk.VirtualMachineFileRestoreGVK,
		TableHeader:  "| Namespace | Name | status.phase | yaml |\n| --- | --- | --- | --- |\n",
		EmptyMessage: "No VirtualMachineFileRestore was found in the cluster",
		ErrorOnEmpty: false,
		RowBuilder: func(item client.Object, filePath string) (string, string) {
			restore := item.(*vmfrv1alpha1.VirtualMachineFileRestore)
			errMsg := ""

			status := ""
			switch phase := restore.Status.Phase; {
			case len(phase) == 0:
				status = "⚠️ no status phase"
				errMsg += fmt.Sprintf("⚠️ VirtualMachineFileRestore **%v** with **no status phase** in **%v** namespace\n\n", restore.Name, restore.Namespace)
			case phase == vmfrv1alpha1.VirtualMachineFileRestorePhaseCompleted:
				status = fmt.Sprintf("✅ status phase %s", phase)
			case slices.Contains([]vmfrv1alpha1.VirtualMachineFileRestorePhase{
				vmfrv1alpha1.VirtualMachineFileRestorePhaseFailed,
				vmfrv1alpha1.VirtualMachineFileRestorePhasePartiallyFailed,
			}, phase):
				status = fmt.Sprintf("❌ status phase %s", phase)
				errMsg += fmt.Sprintf("❌ VirtualMachineFileRestore **%v** with **status phase %s** in **%v** namespace\n\n", restore.Name, phase, restore.Namespace)
			default:
				status = fmt.Sprintf("⚠️ status phase %s", phase)
			}

			link := fmt.Sprintf("[`yaml`](%s)", filePath)
			row := fmt.Sprintf("| %v | %v | %s | %s |\n", restore.Namespace, restore.Name, status, link)
			return row, errMsg
		},
	}, vmFileRestoreList)
}

func ReplaceAvailableStorageClassesSection(outputPath string, storageClassList *storagev1.StorageClassList) {
	ReplaceClusterScopedSection(outputPath, ClusterScopedConfig{
		TemplateKey:  "STORAGE_CLASSES",
		GVK:          gvk.StorageClassGVK,
		FilePath:     "cluster-scoped-resources/storage.k8s.io/storageclasses/storageclasses.yaml",
		EmptyMessage: "No StorageClass was found in the cluster",
	}, storageClassList)
}

func ReplaceAvailableVolumeSnapshotClassesSection(outputPath string, volumeSnapshotClassList *volumesnapshotv1.VolumeSnapshotClassList) {
	ReplaceClusterScopedSection(outputPath, ClusterScopedConfig{
		TemplateKey:  "VOLUME_SNAPSHOT_CLASSES",
		GVK:          gvk.VolumeSnapshotClassGVK,
		FilePath:     "cluster-scoped-resources/snapshot.storage.k8s.io/volumesnapshotclasses/volumesnapshotclasses.yaml",
		EmptyMessage: "No VolumeSnapshotClass was found in the cluster",
	}, volumeSnapshotClassList)
}

func ReplaceAvailableCSIDriversSection(outputPath string, csiDriverList *storagev1.CSIDriverList) {
	ReplaceClusterScopedSection(outputPath, ClusterScopedConfig{
		TemplateKey:  "CSI_DRIVERS",
		GVK:          gvk.CSIDriverGVK,
		FilePath:     "cluster-scoped-resources/storage.k8s.io/csidrivers/csidrivers.yaml",
		EmptyMessage: "No CSIDriver was found in the cluster",
	}, csiDriverList)
}

func ReplaceCustomResourceDefinitionsSection(outputPath string, clusterConfig *rest.Config) {
	errorMessage := "❌ Unable to write CustomResourceDefinitions section: "

	client, err := apiextensionsclientset.NewForConfig(clusterConfig)
	if err != nil {
		summaryTemplateReplaces["ERRORS"] += errorMessage + err.Error() + "\n\n"
		summaryTemplateReplaces["CUSTOM_RESOURCE_DEFINITION"] = errorMessage + err.Error()
		return
	}

	crdsPath := "cluster-scoped-resources/apiextensions.k8s.io/customresourcedefinitions"

	// CRD spec.names.plural : CRD spec.group
	crds := map[string]string{
		"dataprotectionapplications":            gvk.DataProtectionApplicationGVK.Group,
		"dataprotectiontests":                   gvk.DataProtectionTestGVK.Group,
		"cloudstorages":                         gvk.CloudStorageGVK.Group,
		"backupstoragelocations":                gvk.BackupStorageLocationGVK.Group,
		"volumesnapshotlocations":               gvk.VolumeSnapshotLocationGVK.Group,
		"backups":                               gvk.BackupGVK.Group,
		"restores":                              gvk.RestoreGVK.Group,
		"schedules":                             gvk.ScheduleGVK.Group,
		"backuprepositories":                    gvk.BackupRepositoryGVK.Group,
		"datauploads":                           gvk.DataUploadGVK.Group,
		"datadownloads":                         gvk.DataDownloadGVK.Group,
		"podvolumebackups":                      gvk.PodVolumeBackupGVK.Group,
		"podvolumerestores":                     gvk.PodVolumeRestoreGVK.Group,
		"downloadrequests":                      gvk.DownloadRequestGVK.Group,
		"deletebackuprequests":                  gvk.DeleteBackupRequestGVK.Group,
		"serverstatusrequests":                  gvk.ServerStatusRequestGVK.Group,
		"nonadminbackupstoragelocationrequests": gvk.NonAdminBackupStorageLocationRequestGVK.Group,
		"nonadminbackupstoragelocations":        gvk.NonAdminBackupStorageLocationGVK.Group,
		"nonadminbackups":                       gvk.NonAdminBackupGVK.Group,
		"nonadminrestores":                      gvk.NonAdminRestoreGVK.Group,
		"nonadmindownloadrequests":              gvk.NonAdminDownloadRequestGVK.Group,
		"virtualmachinebackupsdiscoveries":      gvk.VirtualMachineBackupsDiscoveryGVK.Group,
		"virtualmachinefilerestores":            gvk.VirtualMachineFileRestoreGVK.Group,
		"clusterserviceversions":                gvk.ClusterServiceVersionGVK.Group,
		"subscriptions":                         gvk.SubscriptionsGVK.Group,
	}

	for crdName, crdGroup := range crds {
		crd, err := client.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), crdName+"."+crdGroup, v1.GetOptions{})
		if err != nil {
			summaryTemplateReplaces["ERRORS"] += errorMessage + err.Error() + "\n\n"
			summaryTemplateReplaces["CUSTOM_RESOURCE_DEFINITION"] += errorMessage + err.Error() + "\n\n"
			continue
		}
		crd.GetObjectKind().SetGroupVersionKind(gvk.CustomResourceDefinitionGVK)
		createYAML(outputPath, crdsPath+fmt.Sprintf("/%s.yaml", crdName), crd)
	}

	summaryTemplateReplaces["CUSTOM_RESOURCE_DEFINITION"] += fmt.Sprintf("For more information, check [`%s`](%s)\n\n", crdsPath, crdsPath)
}

func createYAML(outputPath string, yamlPath string, obj runtime.Object) string {
	objFilePath := outputPath + yamlPath
	dir := path.Dir(objFilePath)
	err := os.MkdirAll(dir, FolderPermission)
	if err != nil {
		return "❌ Unable to create dir " + dir
	}
	result := ""
	newFile, err := os.Create(objFilePath)
	if err != nil {
		fmt.Println(err)
		result = "❌ Unable to create file " + objFilePath
	} else {
		printer := printers.YAMLPrinter{}
		err = printer.PrintObj(obj, newFile)
		if err != nil {
			fmt.Println(err)
			result = "❌ Unable to write " + objFilePath
		} else {
			result = fmt.Sprintf("For more information, check [`%s`](%s)\n\n", yamlPath, yamlPath)
		}
	}
	defer newFile.Close()
	return result
}

func createFile(outputPath string, describePath string, describeOutput string, describeTitle string) string {
	describeFilePath := outputPath + describePath
	dir := path.Dir(describeFilePath)
	err := os.MkdirAll(dir, FolderPermission)
	if err != nil {
		return "❌ Unable to create dir " + dir
	}
	result := ""
	newFile, err := os.Create(describeFilePath)
	if err != nil {
		fmt.Println(err)
		result = "❌ Unable to create file " + describeFilePath
	} else {
		err := os.WriteFile(describeFilePath, []byte(describeOutput), FilePermission)
		if err != nil {
			fmt.Println(err)
			result = "❌ Unable to write " + describeFilePath
		} else {
			result = fmt.Sprintf("[`"+describeTitle+"`](%s)", describePath)
		}
	}
	defer newFile.Close()
	return result
}

func Write(outputPath string) error {
	if len(summaryTemplateReplaces["ERRORS"]) == 0 {
		summaryTemplateReplaces["ERRORS"] += "No errors happened or were found while running OADP must-gather\n\n"
	}

	summary := summaryTemplate
	for _, key := range summaryTemplateReplacesKeys {
		value, ok := summaryTemplateReplaces[key]
		if !ok {
			return fmt.Errorf("key '%s' not set in SummaryTemplateReplaces", key)
		}
		if len(value) == 0 {
			return fmt.Errorf("value for key '%s' not set in SummaryTemplateReplaces", key)
		}
		summary = strings.ReplaceAll(
			summary,
			fmt.Sprintf("<<%s>>", key),
			value,
		)
	}

	summaryPath := outputPath + "oadp-must-gather-summary.md"
	sumary, err := os.Create(summaryPath)
	if err != nil {
		return err
	}
	err = os.WriteFile(summaryPath, []byte(summary), FilePermission)
	if err != nil {
		return err
	}
	defer sumary.Close()
	return nil
}

func WriteVersion(version string) error {
	versionFileContent := fmt.Sprintf(
		`OpenShift API for Data Protection (OADP) Must-gather
%s`,
		version)
	versionFilePath := "must-gather/version"
	versionFile, err := os.Create(versionFilePath)
	if err != nil {
		return err
	}
	err = os.WriteFile(versionFilePath, []byte(versionFileContent), FilePermission)
	if err != nil {
		return err
	}
	defer versionFile.Close()
	return nil
}
