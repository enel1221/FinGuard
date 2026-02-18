package clustercache

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type Cache struct {
	clientset  kubernetes.Interface
	logger     *slog.Logger
	namespaces cache.Store
	nodes      cache.Store
	pods       cache.Store
	stopCh     chan struct{}
	mu         sync.RWMutex
	ready      bool
}

func New(logger *slog.Logger) (*Cache, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			return nil, fmt.Errorf("failed to build k8s config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s clientset: %w", err)
	}

	return NewWithClientset(clientset, logger), nil
}

func NewWithClientset(cs kubernetes.Interface, logger *slog.Logger) *Cache {
	return &Cache{
		clientset: cs,
		logger:    logger,
		stopCh:    make(chan struct{}),
	}
}

func (c *Cache) Start(ctx context.Context) error {
	c.logger.Info("starting cluster cache")

	c.namespaces = c.watchResource(ctx, "namespaces", &corev1.Namespace{}, func(opts metav1.ListOptions) (runtime.Object, error) {
		return c.clientset.CoreV1().Namespaces().List(ctx, opts)
	}, func(opts metav1.ListOptions) (watch.Interface, error) {
		return c.clientset.CoreV1().Namespaces().Watch(ctx, opts)
	})

	c.nodes = c.watchResource(ctx, "nodes", &corev1.Node{}, func(opts metav1.ListOptions) (runtime.Object, error) {
		return c.clientset.CoreV1().Nodes().List(ctx, opts)
	}, func(opts metav1.ListOptions) (watch.Interface, error) {
		return c.clientset.CoreV1().Nodes().Watch(ctx, opts)
	})

	c.pods = c.watchResource(ctx, "pods", &corev1.Pod{}, func(opts metav1.ListOptions) (runtime.Object, error) {
		return c.clientset.CoreV1().Pods("").List(ctx, opts)
	}, func(opts metav1.ListOptions) (watch.Interface, error) {
		return c.clientset.CoreV1().Pods("").Watch(ctx, opts)
	})

	c.mu.Lock()
	c.ready = true
	c.mu.Unlock()
	c.logger.Info("cluster cache ready")
	return nil
}

func (c *Cache) watchResource(
	ctx context.Context,
	name string,
	objType runtime.Object,
	listFunc func(metav1.ListOptions) (runtime.Object, error),
	watchFunc func(metav1.ListOptions) (watch.Interface, error),
) cache.Store {
	store, controller := cache.NewInformer(
		&cache.ListWatch{
			ListFunc:  listFunc,
			WatchFunc: watchFunc,
		},
		objType,
		10*time.Minute,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    func(obj interface{}) { c.logger.Debug("resource added", "type", name) },
			UpdateFunc: func(_, _ interface{}) {},
			DeleteFunc: func(obj interface{}) { c.logger.Debug("resource deleted", "type", name) },
		},
	)

	go controller.Run(ctx.Done())
	return store
}

func (c *Cache) Stop() {
	close(c.stopCh)
}

func (c *Cache) IsReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ready
}

func (c *Cache) GetNamespaces() []*corev1.Namespace {
	if c.namespaces == nil {
		return nil
	}
	items := c.namespaces.List()
	result := make([]*corev1.Namespace, 0, len(items))
	for _, item := range items {
		if ns, ok := item.(*corev1.Namespace); ok {
			result = append(result, ns)
		}
	}
	return result
}

func (c *Cache) GetNodes() []*corev1.Node {
	if c.nodes == nil {
		return nil
	}
	items := c.nodes.List()
	result := make([]*corev1.Node, 0, len(items))
	for _, item := range items {
		if n, ok := item.(*corev1.Node); ok {
			result = append(result, n)
		}
	}
	return result
}

func (c *Cache) GetPods() []*corev1.Pod {
	if c.pods == nil {
		return nil
	}
	items := c.pods.List()
	result := make([]*corev1.Pod, 0, len(items))
	for _, item := range items {
		if p, ok := item.(*corev1.Pod); ok {
			result = append(result, p)
		}
	}
	return result
}

func (c *Cache) GetNamespace(name string) *corev1.Namespace {
	if c.namespaces == nil {
		return nil
	}
	key := name
	item, exists, err := c.namespaces.GetByKey(key)
	if err != nil || !exists {
		return nil
	}
	ns, _ := item.(*corev1.Namespace)
	return ns
}
