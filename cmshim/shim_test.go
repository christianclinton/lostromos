package cmshim

import (
	"fmt"
	"testing"


	"github.com/stretchr/testify/assert"
	"github.com/wpengine/lostromos/printctlr"
	restclient "k8s.io/client-go/rest"
	"github.com/golang/mock/gomock"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	testConfigMap = &v1.ConfigMap{
		ObjectMeta: v12.ObjectMeta{
			SelfLink: "/api/stable.nicolerenee.io/v1/namespaces/mock/characters/dory",
			Namespace: "foo",
			Annotations: map[string]string{
				"com.wpengine.lostromos.crd-type": "stable.nicolerenee.io/v1/Characters",
			},
		},
		TypeMeta: v12.TypeMeta{
			APIVersion: "stable.nicolerenee.io/v1",
			Kind: "Character",
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
			SelfLink: "/api/stable.nicolerenee.io/v1/namespaces/mock/characters/dory",
			Namespace: "foo",
			Annotations: map[string]string{
				"com.wpengine.lostromos.crd-type": "stable.nicolerenee.io/v1/Characters",
			},
		},
		TypeMeta: v12.TypeMeta{
			APIVersion: "stable.nicolerenee.io/v1",
			Kind: "Character",

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
			"kind": "Character",
			"metadata": map[string]interface{}{
				"name": "dory",
				"namespace": "foo",
				"selfLink": "/api/stable.nicolerenee.io/v1/namespaces/mock/characters/dory",
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
			"kind": "Character",
			"metadata": map[string]interface{}{
				"name": "dory",
				"namespace": "foo",
				"selfLink": "/api/stable.nicolerenee.io/v1/namespaces/mock/characters/dory",
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
