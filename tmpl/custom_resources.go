// Copyright 2017 the lostromos Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tmpl

// CustomResource provides some helper methods for interacting with the
// kubernetes custom resource inside the templates.
type CustomResources struct {
	Resources map[string]map[string]*CustomResource // represents the resource from kubernetes
	lastResource *CustomResource // points to the last added resource
}

func NewCustomResources() *CustomResources {
	crs := &CustomResources{
		Resources: make(map[string]map[string]*CustomResource),
	}
	return crs
}

func (crs *CustomResources) AddResource(cr *CustomResource) {
	crKind := cr.Resource.GetAPIVersion() + "/" + cr.Resource.GetKind()
	if _, ok := crs.Resources[crKind]; !ok {
		crs.Resources[crKind] = make(map[string]*CustomResource)
	}
	crs.Resources[crKind][cr.Resource.GetSelfLink()] = cr

	// Emulate legacy behavior if only one CustomResource
	crs.lastResource = cr
}

func (crs *CustomResources) DeleteResource(cr *CustomResource) {
	crKind := cr.Resource.GetAPIVersion() + "/" + cr.Resource.GetKind()
	if _, ok := crs.Resources[crKind]; ok {
		oldResource := crs.Resources[crKind][cr.Resource.GetSelfLink()]
		if &crs.lastResource == &oldResource {
			crs.lastResource = nil
		}
		delete(crs.Resources[crKind], cr.Resource.GetSelfLink())
	}
}

func (crs *CustomResources) GetResources(kind string, apiVersion string) []*CustomResource {
	crKind := apiVersion + "/" + kind
	resources := make([]*CustomResource,0)
	if _, ok := crs.Resources[crKind]; !ok {
		return resources
	}
	for _, resource := range crs.Resources[crKind] {
		resources = append(resources, resource)
	}
	return resources
}

// Emulate legacy behavior if only one CustomResource
func (crs *CustomResources) Name() string {
	if crs.lastResource != nil {
		return crs.lastResource.Name()
	}
	return ""
}


func (crs *CustomResources) GetField(fields ...string) string {
	if crs.lastResource != nil {
		return crs.lastResource.GetField(fields...)
	}
	return ""
}

func (crs *CustomResources) Count() int {
	count := 0
	for _, v := range crs.Resources {
		count += len(v)
	}
	return count
}