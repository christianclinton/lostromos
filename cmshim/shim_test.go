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

package cmshim

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/wpengine/lostromos/printctlr"
	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	restclient "k8s.io/client-go/rest"
)

var (
	testConfigMap = &v1.ConfigMap{
		ObjectMeta: v12.ObjectMeta{
			SelfLink:  "/api/stable.nicolerenee.io/v1/namespaces/mock/characters/dory",
			Namespace: "foo",
			Annotations: map[string]string{
				"com.wpengine.lostromos.crd-type": "stable.nicolerenee.io/v1/Characters",
			},
		},
		TypeMeta: v12.TypeMeta{
			APIVersion: "stable.nicolerenee.io/v1",
			Kind:       "Character",
		},
		Data: map[string]string{
			"crd": `---
apiVersion: stable.nicolerenee.io/v1
kind: Character
metadata:
  name: dory
spec:
  name: Dory
  from: "Finding Nemo"
  by: Disney`,
		},
	}
	testConfigMapUpdated = &v1.ConfigMap{
		ObjectMeta: v12.ObjectMeta{
			SelfLink:  "/api/stable.nicolerenee.io/v1/namespaces/mock/characters/dory",
			Namespace: "foo",
			Annotations: map[string]string{
				"com.wpengine.lostromos.crd-type": "stable.nicolerenee.io/v1/Characters",
			},
		},
		TypeMeta: v12.TypeMeta{
			APIVersion: "stable.nicolerenee.io/v1",
			Kind:       "Character",
		},
		Data: map[string]string{
			"crd": `---
apiVersion: stable.nicolerenee.io/v1
kind: Character
metadata:
  name: dory
spec:
  name: Dory2
  from: "Finding Nemo"
  by: Disney`,
		},
	}
	testResource = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "stable.nicolerenee.io/v1",
			"kind":       "Character",
			"metadata": map[string]interface{}{
				"name":      "dory",
				"namespace": "foo",
				"selfLink":  "/api/stable.nicolerenee.io/v1/namespaces/mock/characters/dory",
			},
			"spec": map[string]interface{}{
				"name": "Dory",
				"from": "Finding Nemo",
				"by":   "Disney",
			},
		},
	}
	testResourceUpdated = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "stable.nicolerenee.io/v1",
			"kind":       "Character",
			"metadata": map[string]interface{}{
				"name":      "dory",
				"namespace": "foo",
				"selfLink":  "/api/stable.nicolerenee.io/v1/namespaces/mock/characters/dory",
			},
			"spec": map[string]interface{}{
				"name": "Dory2",
				"from": "Finding Nemo",
				"by":   "Disney",
			},
		},
	}
)

type logResult struct {
	msg string
}

type testLogger struct {
	res *logResult
}

func (c testLogger) Error(err error) {
	c.res.msg = fmt.Sprintf("error: %s", err)
}

func TestNewCRWatcher(t *testing.T) {
	kubeCfg := &restclient.Config{}
	cfg := &Config{}

	cw, err := NewCMShim(cfg, kubeCfg, printctlr.Controller{}, testLogger{})

	assert.Nil(t, err)
	assert.Equal(t, cfg, cw.Config)
	assert.NotNil(t, cw.handler)
	assert.NotNil(t, cw.controller)
	assert.NotNil(t, cw.logger)
}

func TestSetupHandlerAddFunc(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockRC := NewMockResourceController(mockCtrl)
	cw := &CMShim{
		Config: &Config{
			CRDType: "stable.nicolerenee.io/v1/Characters",
		},
	}
	cw.setupHandler(mockRC)

	mockRC.EXPECT().ResourceAdded(testResource)

	cw.handler.OnAdd(testConfigMap)
}

func TestSetupHandlerDeleteFunc(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockRC := NewMockResourceController(mockCtrl)
	cw := &CMShim{
		Config: &Config{
			CRDType: "stable.nicolerenee.io/v1/Characters",
		},
	}
	cw.setupHandler(mockRC)

	mockRC.EXPECT().ResourceDeleted(testResource)

	cw.handler.OnDelete(testConfigMap)
}

func TestSetupHandlerUpdateFunc(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockRC := NewMockResourceController(mockCtrl)
	cw := &CMShim{
		Config: &Config{
			CRDType: "stable.nicolerenee.io/v1/Characters",
		},
	}

	cw.setupHandler(mockRC)

	mockRC.EXPECT().ResourceUpdated(testResource, testResourceUpdated)

	cw.handler.OnUpdate(testConfigMap, testConfigMapUpdated)
}
