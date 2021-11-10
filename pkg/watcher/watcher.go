package watcher

import (
	"context"
	"strings"
	"sync"

	"github.com/rancher/lasso/pkg/dynamic"
	"github.com/rancher/wrangler/pkg/clients"
	"github.com/rancher/wrangler/pkg/kv"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

type Watcher struct {
	mapper   meta.RESTMapper
	cdi      discovery.CachedDiscoveryInterface
	dynamic  *dynamic.Controller
	matchers []dynamic.GVKMatcher
}

func New(clients *clients.Clients) (*Watcher, error) {
	mapper, err := clients.ToRESTMapper()
	if err != nil {
		return nil, err
	}

	cdi, err := clients.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		mapper:  mapper,
		cdi:     cdi,
		dynamic: clients.Dynamic,
	}, nil
}

func (w *Watcher) isListWatchable(gvk schema.GroupVersionKind) bool {
	mapping, err := w.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return false
	}

	_, resources, err := w.cdi.ServerGroupsAndResources()
	if err != nil {
		return false
	}

	for _, res := range resources {
		if res.GroupVersion == gvk.GroupVersion().String() {
			for _, apiResource := range res.APIResources {
				if apiResource.Name == mapping.Resource.Resource {
					list := false
					watch := false
					for _, verb := range apiResource.Verbs {
						if verb == "list" {
							list = true
						} else if verb == "watch" {
							watch = true
						}
					}
					return list && watch
				}
			}
		}
	}

	return false
}

func (w *Watcher) shouldWatch(gvk schema.GroupVersionKind) bool {
	if !w.isListWatchable(gvk) {
		return false
	}

	if len(w.matchers) == 0 {
		return true
	}

	for _, matcher := range w.matchers {
		if matcher(gvk) {
			return true
		}
	}

	return false
}

func (w *Watcher) Start(ctx context.Context) (chan runtime.Object, error) {
	var chanLock sync.Mutex

	result := make(chan runtime.Object, 100)
	go func() {
		<-ctx.Done()
		chanLock.Lock()
		close(result)
		result = nil
		chanLock.Unlock()
	}()

	w.dynamic.OnChange(ctx, "watcher", w.shouldWatch, func(obj runtime.Object) (runtime.Object, error) {
		chanLock.Lock()
		defer chanLock.Unlock()
		if obj != nil && result != nil {
			result <- obj
		}
		return obj, nil
	})

	return result, nil
}

func (w *Watcher) MatchName(name string) {
	w.matchers = append(w.matchers, func(gvk schema.GroupVersionKind) bool {
		return w.isName(name, gvk)
	})
}

func (w *Watcher) isName(name string, gvk schema.GroupVersionKind) bool {
	mapping, err := w.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return false
	}

	resource, group := kv.Split(name, ".")
	return (resource == "*" || mapping.Resource.Resource == resource || strings.ToLower(gvk.Kind) == resource) &&
		gvk.Group == group
}
