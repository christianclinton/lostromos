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
	"errors"
	"time"

	"github.com/wpengine/lostromos/crwatcher"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	yaml2 "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const CRD_ANNOTATION = "com.wpengine.lostromos.crd-type"

// Config provides config for a ConfigMap Shim
type Config struct {
	CRDType string        // CRD type annotation to watch
	Resync  time.Duration // How often existing ConfigMaps should be resynced (marked as updated)
}

// Watches ConfigMaps with a CRD payload.
type CMShim struct {
	Config             *Config
	handler            cache.ResourceEventHandlerFuncs
	controller         cache.SharedIndexInformer
	logger             ErrorLogger
	resourceController crwatcher.ResourceController
}

// ErrorLogger will receive any error messages from the kubernetes client
type ErrorLogger interface {
	Error(err error)
}

// NewCMShim builds a CMShim
func NewCMShim(cfg *Config, kubeCfg *restclient.Config, rc crwatcher.ResourceController, l ErrorLogger) (*CMShim, error) {
	cw := &CMShim{
		Config:             cfg,
		logger:             l,
		resourceController: rc,
	}

	client := kubernetes.NewForConfigOrDie(kubeCfg)
	sharedInformers := informers.NewSharedInformerFactory(client, cw.Config.Resync)
	cw.controller = sharedInformers.Core().V1().ConfigMaps().Informer()

	cw.setupHandler(rc)
	cw.controller.AddEventHandler(cw.handler)
	cw.setupRuntimeLogging()
	return cw, nil
}

// Takes a ConfigMap with a `crd` Data field containing a CRD YAML payload
// and returns an Unstructured object.
func configMapToCRD(cm *v1.ConfigMap) (*unstructured.Unstructured, error) {
	crdUnstructured := &unstructured.Unstructured{}

	if _, ok := cm.Data["crd"]; !ok {
		return nil, errors.New("Missing `crd` field in `Data` of ConfigMap")
	}

	crdYAML := []byte(cm.Data["crd"])

	crdJSON, err := yaml2.ToJSON(crdYAML)
	if err != nil {
		return nil, err
	}

	err = crdUnstructured.UnmarshalJSON(crdJSON)
	if err != nil {
		return nil, err
	}

	// Copy ID fields from ConfigMap to fake CRD
	crdUnstructured.SetSelfLink(cm.SelfLink)
	crdUnstructured.SetNamespace(cm.Namespace)
	return crdUnstructured, nil
}

func (cw *CMShim) setupRuntimeLogging() {
	if cw.logger != nil {
		utilruntime.ErrorHandlers = []func(error){
			cw.logKubeError,
		}
	}
}

func (cw *CMShim) logKubeError(err error) {
	cw.logger.Error(err)
}

func (cw *CMShim) setupHandler(con crwatcher.ResourceController) {
	cw.handler = cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			r := obj.(*v1.ConfigMap)
			if !cw.passesFiltering(r) {
				return
			}
			crd, err := configMapToCRD(r)
			if err != nil {
				cw.logKubeError(err)
				return
			}
			con.ResourceAdded(crd)
		},
		DeleteFunc: func(obj interface{}) {
			r := obj.(*v1.ConfigMap)
			if !cw.passesFiltering(r) {
				return
			}
			crd, err := configMapToCRD(r)
			if err != nil {
				cw.logKubeError(err)
				return
			}
			con.ResourceDeleted(crd)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldR := oldObj.(*v1.ConfigMap)
			newR := newObj.(*v1.ConfigMap)

			cw.update(con, oldR, newR)
		},
	}
}

// update sends an appropriate notification to the controller based on filtering outcomes of the old and new state of a
// resource.
//
// If no filter is configured or both states of the resource pass filtering, send an update to the controller.
// If the new state passes filtering and the old state does not, send an add notification to the controller.
// If the old state passes filtering and the new state does not, send a delete notification to the controller.
// If neither state passes filtering, ignore.
//
func (cw *CMShim) update(con crwatcher.ResourceController, oldR *v1.ConfigMap, newR *v1.ConfigMap) {
	var (
		oldCRD *unstructured.Unstructured
		newCRD *unstructured.Unstructured
		err    error
	)
	// Convert ConfigMap if annotated as a CRD.
	if cw.passesFiltering(oldR) {
		oldCRD, err = configMapToCRD(oldR)
		if err != nil {
			cw.logKubeError(err)
			return
		}
	}
	if cw.passesFiltering(newR) {
		newCRD, err = configMapToCRD(newR)
		if err != nil {
			cw.logKubeError(err)
			return
		}
	}
	// Perform Add/Update/Delete based on annotation combination
	if cw.passesFiltering(newR) {
		if cw.passesFiltering(oldR) {
			con.ResourceUpdated(oldCRD, newCRD)
		} else {
			con.ResourceAdded(newCRD)
		}
	} else if cw.passesFiltering(oldR) {
		con.ResourceDeleted(oldCRD)
	}
}

// passesFiltering checks if the ConfigMap is annotated with a CRD type.
// This indicates the ConfigMap has a CRD payload to extract.
func (cw *CMShim) passesFiltering(r *v1.ConfigMap) bool {
	annotations := r.GetAnnotations()
	if annotations == nil {
		return false
	}

	if _, ok := annotations[CRD_ANNOTATION]; !ok {
		return false
	}

	return annotations[CRD_ANNOTATION] == cw.Config.CRDType
}

// Watch will be called to begin watching the configured custom resource. All
// events will be passed back to the ResourceController
func (cw *CMShim) Watch(stopCh <-chan struct{}) error {
	if cw.controller == nil {
		return errors.New("the CMShim has not been initialized")
	}

	// Kick off wait for cache sync.
	// This will return a few moments after the Informer Run() loop
	// starts after this.
	go func() {
		cache.WaitForCacheSync(stopCh, cw.controller.HasSynced)
		cw.resourceController.NotifySynced()
	}()

	cw.controller.Run(stopCh)

	return nil
}
