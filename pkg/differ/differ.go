package differ

import (
	"encoding/json"
	"fmt"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/rancher/wrangler/pkg/clients"
	"github.com/rancher/wrangler/pkg/gvk"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

type Differ struct {
	cache  map[string]runtime.Object
	mapper meta.RESTMapper
}

func New(clients *clients.Clients) (*Differ, error) {
	mapper, err := clients.ToRESTMapper()
	if err != nil {
		return nil, err
	}
	return &Differ{
		cache:  map[string]runtime.Object{},
		mapper: mapper,
	}, nil
}

func (d *Differ) Print(obj runtime.Object) error {
	key, err := key(obj)
	if err != nil {
		return err
	}

	meta, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	old, ok := d.cache[key]
	d.cache[key] = obj

	if !ok || meta.GetResourceVersion() == "" {
		return nil
	}

	oldBytes, err := toBytes(old)
	if err != nil {
		return err
	}

	newBytes, err := toBytes(obj)
	if err != nil {
		return err
	}

	patch, err := jsonpatch.CreateMergePatch(oldBytes, newBytes)
	if err != nil {
		return err
	}

	printKey, err := printKey(obj, d.mapper)
	if err != nil {
		return err
	}

	if string(patch) != "{}" {
		fmt.Printf("%s %s %s\n", meta.GetResourceVersion(), printKey, patch)
	}

	return nil
}

func clearRevision(obj runtime.Object) (runtime.Object, error) {
	obj = obj.DeepCopyObject()
	meta, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	meta.SetResourceVersion("")
	return obj, nil
}

func toBytes(obj runtime.Object) ([]byte, error) {
	obj, err := clearRevision(obj)
	if err != nil {
		return nil, err
	}

	return json.Marshal(obj)
}

func printKey(obj runtime.Object, mapper meta.RESTMapper) (string, error) {
	gvk, err := gvk.Get(obj)
	if err != nil {
		return "", err
	}

	meta, err := meta.Accessor(obj)
	if err != nil {
		return "", err
	}

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return "", err
	}

	buf := &strings.Builder{}
	buf.WriteString(mapping.Resource.Resource)
	if gvk.Group != "" {
		buf.WriteString(".")
		buf.WriteString(gvk.Group)
	}
	buf.WriteString(" ")
	if meta.GetNamespace() != "" {
		buf.WriteString(meta.GetNamespace())
		buf.WriteString("/")
	}
	buf.WriteString(meta.GetName())
	return buf.String(), nil
}

func key(obj runtime.Object) (string, error) {
	meta, err := meta.Accessor(obj)
	if err != nil {
		return "", err
	}
	id := strings.Builder{}
	if meta.GetNamespace() != "" {
		id.WriteString(meta.GetNamespace())
		id.WriteString("/")
	}
	id.WriteString(meta.GetName())
	id.WriteString(" ")
	gvk, err := gvk.Get(obj)
	if err != nil {
		return "", err
	}
	id.WriteString(gvk.String())
	return id.String(), nil
}
