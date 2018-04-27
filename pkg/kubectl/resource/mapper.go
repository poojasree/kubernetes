/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resource

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Mapper is a convenience struct for holding references to the interfaces
// needed to create Info for arbitrary objects.
type Mapper struct {
	RESTMapper   meta.RESTMapper
	ClientMapper ClientMapper
	Decoder      runtime.Decoder
}

// InfoForData creates an Info object for the given data. An error is returned
// if any of the decoding or client lookup steps fail. Name and namespace will be
// set into Info if the mapping's MetadataAccessor can retrieve them.
func (m *Mapper) InfoForData(data []byte, source string) (*Info, error) {
	obj, gvk, err := m.Decoder.Decode(data, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to decode %q: %v", source, err)
	}

	mapping, err := m.RESTMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("unable to recognize %q: %v", source, err)
	}

	client, err := m.ClientMapper.ClientForMapping(mapping)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to a server to handle %q: %v", mapping.Resource, err)
	}

	name, _ := metadataAccessor.Name(obj)
	namespace, _ := metadataAccessor.Namespace(obj)
	resourceVersion, _ := metadataAccessor.ResourceVersion(obj)

	return &Info{
		Client:  client,
		Mapping: mapping,

		Source:          source,
		Namespace:       namespace,
		Name:            name,
		ResourceVersion: resourceVersion,

		Object: obj,
	}, nil
}

// InfoForObject creates an Info object for the given Object. An error is returned
// if the object cannot be introspected. Name and namespace will be set into Info
// if the mapping's MetadataAccessor can retrieve them.
func (m *Mapper) InfoForObject(obj runtime.Object, typer runtime.ObjectTyper, preferredGVKs []schema.GroupVersionKind) (*Info, error) {
	groupVersionKinds, _, err := typer.ObjectKinds(obj)
	if err != nil {
		return nil, fmt.Errorf("unable to get type info from the object %q: %v", reflect.TypeOf(obj), err)
	}

	groupVersionKind := groupVersionKinds[0]
	if len(groupVersionKinds) > 1 && len(preferredGVKs) > 0 {
		groupVersionKind = preferredObjectKind(groupVersionKinds, preferredGVKs)
	}

	mapping, err := m.RESTMapper.RESTMapping(groupVersionKind.GroupKind(), groupVersionKind.Version)
	if err != nil {
		return nil, fmt.Errorf("unable to recognize %v: %v", groupVersionKind, err)
	}

	client, err := m.ClientMapper.ClientForMapping(mapping)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to a server to handle %q: %v", mapping.Resource, err)
	}
	name, _ := metadataAccessor.Name(obj)
	namespace, _ := metadataAccessor.Namespace(obj)
	resourceVersion, _ := metadataAccessor.ResourceVersion(obj)
	return &Info{
		Client:  client,
		Mapping: mapping,

		Namespace:       namespace,
		Name:            name,
		ResourceVersion: resourceVersion,

		Object: obj,
	}, nil
}

// preferredObjectKind picks the possibility that most closely matches the priority list in this order:
// GroupVersionKind matches (exact match)
// GroupKind matches
// Group matches
func preferredObjectKind(possibilities []schema.GroupVersionKind, preferences []schema.GroupVersionKind) schema.GroupVersionKind {
	// Exact match
	for _, priority := range preferences {
		for _, possibility := range possibilities {
			if possibility == priority {
				return possibility
			}
		}
	}

	// GroupKind match
	for _, priority := range preferences {
		for _, possibility := range possibilities {
			if possibility.GroupKind() == priority.GroupKind() {
				return possibility
			}
		}
	}

	// Group match
	for _, priority := range preferences {
		for _, possibility := range possibilities {
			if possibility.Group == priority.Group {
				return possibility
			}
		}
	}

	// Just pick the first
	return possibilities[0]
}

// DisabledClientForMapping allows callers to avoid allowing remote calls when handling
// resources.
type DisabledClientForMapping struct {
	ClientMapper
}

func (f DisabledClientForMapping) ClientForMapping(mapping *meta.RESTMapping) (RESTClient, error) {
	return nil, nil
}
