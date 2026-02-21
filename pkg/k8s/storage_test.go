package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListPVCs(t *testing.T) {
	storageClass := "standard"
	fakeClient := fake.NewSimpleClientset(
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "data-pvc", Namespace: "default"},
			Spec: corev1.PersistentVolumeClaimSpec{
				StorageClassName: &storageClass,
				AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			},
			Status: corev1.PersistentVolumeClaimStatus{
				Phase: corev1.ClaimBound,
				Capacity: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	pvcs, err := client.ListPVCs(context.Background(), "default", metav1.ListOptions{})
	if err != nil {
		t.Fatalf("ListPVCs() error = %v", err)
	}
	if len(pvcs) != 1 {
		t.Errorf("expected 1 PVC, got %d", len(pvcs))
	}
	if pvcs[0].Status.Phase != corev1.ClaimBound {
		t.Errorf("expected phase Bound, got %s", pvcs[0].Status.Phase)
	}
}

func TestListPVs(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(
		&corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{Name: "pv-1"},
			Spec: corev1.PersistentVolumeSpec{
				Capacity: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("100Gi"),
				},
				PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
				StorageClassName:              "fast",
			},
			Status: corev1.PersistentVolumeStatus{
				Phase: corev1.VolumeBound,
			},
		},
	)

	client := NewClusterClientForTesting(fakeClient, nil)

	pvs, err := client.ListPVs(context.Background())
	if err != nil {
		t.Fatalf("ListPVs() error = %v", err)
	}
	if len(pvs) != 1 {
		t.Errorf("expected 1 PV, got %d", len(pvs))
	}
}
