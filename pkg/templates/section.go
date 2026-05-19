package templates

import (
	"fmt"

	"github.com/openshift/oadp-must-gather/pkg/gvk"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SectionConfig struct {
	TemplateKey  string                  // "BACKUPS"
	APIGroup     string                  // "velero.io"
	ResourceName string                  // "backups"
	GVK          schema.GroupVersionKind // gvk.BackupGVK
	TableHeader  string                  // "| Namespace | Name | yaml |..."
	EmptyMessage string                  // "No Backup was found in the cluster"
	ErrorOnEmpty bool                    // Write to ERRORS if true
	RowBuilder   RowBuilder
}

type RowBuilder func(item client.Object, filePath string) (row string, errMsg string)

func ReplaceSection(outputPath string, sectionConfig SectionConfig, list client.ObjectList) {

	handleEmpty := func() {
		summaryTemplateReplaces[sectionConfig.TemplateKey] = "❌ " + sectionConfig.EmptyMessage
		if sectionConfig.ErrorOnEmpty {
			summaryTemplateReplaces["ERRORS"] += "⚠️ " + sectionConfig.EmptyMessage + "\n\n"
		}
	}

	if list == nil {
		handleEmpty()
		return
	}

	items, err := meta.ExtractList(list)

	if err != nil || len(items) == 0 {
		handleEmpty()
		return
	}

	summaryTemplateReplaces[sectionConfig.TemplateKey] += sectionConfig.TableHeader

	itemsByNamespace := map[string][]client.Object{}
	for _, item := range items {
		obj := item.(client.Object)
		itemsByNamespace[obj.GetNamespace()] = append(itemsByNamespace[obj.GetNamespace()], obj)
	}

	for namespace, nsItems := range itemsByNamespace {
		list := &corev1.List{}
		list.GetObjectKind().SetGroupVersionKind(gvk.ListGVK)

		folder := fmt.Sprintf("namespaces/%s/%s/%s", namespace, sectionConfig.APIGroup, sectionConfig.ResourceName)
		file := folder + "/" + sectionConfig.ResourceName + ".yaml"

		for _, obj := range nsItems {
			obj.GetObjectKind().SetGroupVersionKind(sectionConfig.GVK)
			list.Items = append(list.Items, runtime.RawExtension{Object: obj})

			row, errMsg := sectionConfig.RowBuilder(obj, file)
			summaryTemplateReplaces[sectionConfig.TemplateKey] += row
			if errMsg != "" {
				summaryTemplateReplaces["ERRORS"] += errMsg
			}
		}
      	createYAML(outputPath, file, list)
  	}
}

type ClusterScopedConfig struct {
	TemplateKey  string
	GVK          schema.GroupVersionKind
	FilePath     string // e.g. "cluster-scoped-resources/storage.k8s.io/storageclasses/storageclasses.yaml"
	EmptyMessage string
}

func ReplaceClusterScopedSection(outputPath string, config ClusterScopedConfig, list client.ObjectList) {
	handleEmpty := func() {
		summaryTemplateReplaces[config.TemplateKey] = "❌ " + config.EmptyMessage
		summaryTemplateReplaces["ERRORS"] += "⚠️ " + config.EmptyMessage + "\n\n"
	}

	if list == nil {
		handleEmpty()
		return
	}

	items, err := meta.ExtractList(list)
	if err != nil || len(items) == 0 {
		handleEmpty()
		return
	}

	corev1List := &corev1.List{}
	corev1List.GetObjectKind().SetGroupVersionKind(gvk.ListGVK)

	for _, item := range items {
		obj := item.(client.Object)
		obj.GetObjectKind().SetGroupVersionKind(config.GVK)
		corev1List.Items = append(corev1List.Items, runtime.RawExtension{Object: obj})
	}

	summaryTemplateReplaces[config.TemplateKey] = createYAML(outputPath, config.FilePath, corev1List)
}
