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

package tmpl_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wpengine/lostromos/tmpl"
)

func TestAddResource(t *testing.T) {
	crs := tmpl.NewCustomResources()
	crs.AddResource(testCR)
	assert.Equal(t, "dory", crs.Name())
}

func TestDeleteResource(t *testing.T) {
	crs := tmpl.NewCustomResources()
	crs.DeleteResource(testCR)
	assert.Equal(t, "", crs.Name())
}

func TestGetResources(t *testing.T) {
	crs := tmpl.NewCustomResources()
	crs.AddResource(testCR)
	resources := crs.GetResources(testCR.Resource.GetKind(), testCR.Resource.GetAPIVersion())
	assert.Equal(t, "dory", resources[0].Name())
}