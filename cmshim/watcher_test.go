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
	cfg := &Config{PluralName: "test"}

	cw, err := NewCMShim(cfg, kubeCfg, printctlr.Controller{}, testLogger{})

	assert.Nil(t, err)
	assert.Equal(t, cfg, cw.Config)
	//assert.NotNil(t, cw.resource)
	assert.NotNil(t, cw.handler)
	//assert.NotNil(t, cw.store)
	assert.NotNil(t, cw.controller)
	assert.NotNil(t, cw.logger)
}

func TestSetupHandlerAddFunc(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockRC := NewMockResourceController(mockCtrl)
	cw := &CMShim{
		Config: &Config{},
	}
	cm := &v1.ConfigMap{
		Data: map[string]string{
			"crd": `---
apiVersion: octolog.github.com/v1alpha1
kind: OctologRateLimit
metadata:
  name: Thing1`,
		},
	}
	crd := &unstructured.Unstructured{}
	crd.SetKind("OctologRateLimit")
	crd.SetAPIVersion("octolog.github.com/v1alpha1")
	crd.SetName("Thing1")

	cw.setupHandler(mockRC)

	mockRC.EXPECT().ResourceAdded(crd)

	cw.handler.OnAdd(cm)
}