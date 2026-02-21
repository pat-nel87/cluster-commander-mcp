package util

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// ListOptions builds metav1.ListOptions from label and field selectors.
func ListOptions(labelSelector, fieldSelector string) metav1.ListOptions {
	opts := metav1.ListOptions{}
	if labelSelector != "" {
		opts.LabelSelector = labelSelector
	}
	if fieldSelector != "" {
		opts.FieldSelector = fieldSelector
	}
	return opts
}

// NamespaceOrAll returns the namespace or empty string for all namespaces.
func NamespaceOrAll(ns string) string {
	if ns == "all" || ns == "*" || ns == "" {
		return ""
	}
	return ns
}
