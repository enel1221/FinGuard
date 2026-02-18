package clustercache

import (
	"log/slog"
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestCache_GetNamespaces(t *testing.T) {
	cs := fake.NewSimpleClientset()
	c := NewWithClientset(cs, testLogger())

	store := cache.NewStore(cache.MetaNamespaceKeyFunc)
	store.Add(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default", Labels: map[string]string{"team": "platform"}}})
	store.Add(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system"}})
	c.namespaces = store

	namespaces := c.GetNamespaces()
	if len(namespaces) != 2 {
		t.Fatalf("expected 2 namespaces, got %d", len(namespaces))
	}
}

func TestCache_GetNodes(t *testing.T) {
	cs := fake.NewSimpleClientset()
	c := NewWithClientset(cs, testLogger())

	store := cache.NewStore(cache.MetaNamespaceKeyFunc)
	store.Add(&corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "node-1",
			Labels: map[string]string{"topology.kubernetes.io/region": "us-east-1"},
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("16Gi"),
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("3800m"),
				corev1.ResourceMemory: resource.MustParse("15Gi"),
			},
		},
	})
	c.nodes = store

	nodes := c.GetNodes()
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Name != "node-1" {
		t.Errorf("expected 'node-1', got %q", nodes[0].Name)
	}
	if nodes[0].Labels["topology.kubernetes.io/region"] != "us-east-1" {
		t.Error("expected region label")
	}
}

func TestCache_GetPods(t *testing.T) {
	cs := fake.NewSimpleClientset()
	c := NewWithClientset(cs, testLogger())

	store := cache.NewStore(cache.MetaNamespaceKeyFunc)
	store.Add(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "default"}})
	store.Add(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-2", Namespace: "default"}})
	c.pods = store

	pods := c.GetPods()
	if len(pods) != 2 {
		t.Fatalf("expected 2 pods, got %d", len(pods))
	}
}

func TestCache_GetNamespace(t *testing.T) {
	cs := fake.NewSimpleClientset()
	c := NewWithClientset(cs, testLogger())

	store := cache.NewStore(cache.MetaNamespaceKeyFunc)
	store.Add(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default", Labels: map[string]string{"cost-center": "eng"}}})
	c.namespaces = store

	ns := c.GetNamespace("default")
	if ns == nil {
		t.Fatal("expected to find namespace")
	}
	if ns.Labels["cost-center"] != "eng" {
		t.Errorf("expected cost-center 'eng', got %q", ns.Labels["cost-center"])
	}

	missing := c.GetNamespace("nonexistent")
	if missing != nil {
		t.Error("expected nil for nonexistent namespace")
	}
}

func TestCache_NilStores(t *testing.T) {
	cs := fake.NewSimpleClientset()
	c := NewWithClientset(cs, testLogger())

	if ns := c.GetNamespaces(); ns != nil {
		t.Error("expected nil for uninitialized namespaces")
	}
	if nodes := c.GetNodes(); nodes != nil {
		t.Error("expected nil for uninitialized nodes")
	}
	if pods := c.GetPods(); pods != nil {
		t.Error("expected nil for uninitialized pods")
	}
	if ns := c.GetNamespace("any"); ns != nil {
		t.Error("expected nil for uninitialized namespace lookup")
	}
}

func TestCache_IsReady(t *testing.T) {
	cs := fake.NewSimpleClientset()
	c := NewWithClientset(cs, testLogger())

	if c.IsReady() {
		t.Error("cache should not be ready before Start")
	}

	c.mu.Lock()
	c.ready = true
	c.mu.Unlock()

	if !c.IsReady() {
		t.Error("cache should be ready after setting flag")
	}
}
