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

package cmd

import (
	"errors"
	"path/filepath"

	"net/http"

	"github.com/mitchellh/go-homedir"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wpengine/lostromos/cmshim"
	"github.com/wpengine/lostromos/crwatcher"
	"github.com/wpengine/lostromos/helmctlr"
	"github.com/wpengine/lostromos/printctlr"
	"github.com/wpengine/lostromos/status"
	"github.com/wpengine/lostromos/tmplctlr"
	"github.com/wpengine/lostromos/version"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: `Start the server.`,
	Run: func(command *cobra.Command, args []string) {
		if err := startServer(); err != nil {
			logger.Errorw("failed to start server", "error", err)
		}
	},
}

func init() {
	LostromosCmd.AddCommand(startCmd)

	startCmd.Flags().String("cm-type", "", "The CRD type to listen for, specified by the annotation: "+cmshim.CRD_ANNOTATION)
	startCmd.Flags().String("cm-namespace", "", "(optional) the namespace of the ConfigMaps you want monitored (either cm-namespace, or cm-all-namespaces must be specified)")
	startCmd.Flags().Bool("cm-all-namespaces", false, "(optional) listen for ConfigMaps from all namespaces (either cm-namespace, or cm-all-namespaces must be specified)")
	startCmd.Flags().String("crd-name", "", "the plural name of the CRD you want monitored (ex: users)")
	startCmd.Flags().String("crd-group", "", "the group of the CRD you want monitored (ex: stable.wpengine.io)")
	startCmd.Flags().String("crd-version", "v1", "the version of the CRD you want monitored")
	startCmd.Flags().String("crd-namespace", metav1.NamespaceNone, "(optional) the namespace of the CRD you want monitored, only needed for namespaced CRDs (ex: default)")
	startCmd.Flags().String("crd-filter", "", "(optional) Annotation key to specify that the custom resource has opted in to watching by Lostromos")
	startCmd.Flags().String("helm-chart", "", "Path for helm chart")
	startCmd.Flags().String("helm-ns", "default", "Namespace for resources deployed by helm")
	startCmd.Flags().String("helm-prefix", "lostromos", "Prefix for release names in helm")
	startCmd.Flags().String("helm-tiller", "tiller-deploy:44134", "Address for helm tiller")
	startCmd.Flags().Bool("helm-wait", false, "Use the helm --wait flag for creating and updating releases")
	startCmd.Flags().Int64("helm-wait-timeout", 120, "The time in seconds to wait for kubernetes resources to be created when doing a helm install or upgrade")
	startCmd.Flags().String("kube-config", filepath.Join(homeDir(), ".kube", "config"), "absolute path to the kubeconfig file. Only required if running outside-of-cluster.")
	startCmd.Flags().Bool("nop", false, "nop")
	startCmd.Flags().String("server-address", ":8080", "The address and port for endpoints such as /metrics and /status")
	startCmd.Flags().String("metrics-endpoint", "/metrics", "The URI for the metrics endpoint")
	startCmd.Flags().String("status-endpoint", "/status", "The URI for the status endpoint")
	startCmd.Flags().String("templates", "", "absolute path to the directory with your template files")

	viperBindFlag("cm.type", startCmd.Flags().Lookup("cm-type"))
	viperBindFlag("cm.namespace", startCmd.Flags().Lookup("cm-namespace"))
	viperBindFlag("cm.allNamespaces", startCmd.Flags().Lookup("cm-all-namespaces"))
	viperBindFlag("crd.name", startCmd.Flags().Lookup("crd-name"))
	viperBindFlag("crd.group", startCmd.Flags().Lookup("crd-group"))
	viperBindFlag("crd.version", startCmd.Flags().Lookup("crd-version"))
	viperBindFlag("crd.namespace", startCmd.Flags().Lookup("crd-namespace"))
	viperBindFlag("crd.filter", startCmd.Flags().Lookup("crd-filter"))
	viperBindFlag("helm.chart", startCmd.Flags().Lookup("helm-chart"))
	viperBindFlag("helm.namespace", startCmd.Flags().Lookup("helm-ns"))
	viperBindFlag("helm.releasePrefix", startCmd.Flags().Lookup("helm-prefix"))
	viperBindFlag("helm.tiller", startCmd.Flags().Lookup("helm-tiller"))
	viperBindFlag("helm.wait", startCmd.Flags().Lookup("helm-wait"))
	viperBindFlag("helm.waitTimeout", startCmd.Flags().Lookup("helm-wait-timeout"))
	viperBindFlag("k8s.config", startCmd.Flags().Lookup("kube-config"))
	viperBindFlag("nop", startCmd.Flags().Lookup("nop"))
	viperBindFlag("server.address", startCmd.Flags().Lookup("server-address"))
	viperBindFlag("server.metricsEndpoint", startCmd.Flags().Lookup("metrics-endpoint"))
	viperBindFlag("server.statusEndpoint", startCmd.Flags().Lookup("status-endpoint"))
	viperBindFlag("templates", startCmd.Flags().Lookup("templates"))
}

func homeDir() string {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	return home
}

func getKubeClient() (*restclient.Config, error) {
	var (
		cfg *restclient.Config
		err error
	)

	cfg, err = restclient.InClusterConfig()
	if err == nil {
		viper.Set("k8s.config", "")
		return cfg, err
	}

	return clientcmd.BuildConfigFromFlags("", viper.GetString("k8s.config"))
}

func buildCMShim(cfg *restclient.Config) (*cmshim.CMShim, error) {
	cmCfg := &cmshim.Config{
		CRDType: viper.GetString("cm.type"),
	}
	ctlr := getController()
	l := &crLogger{logger: logger}
	return cmshim.NewCMShim(cmCfg, cfg, ctlr, l)
}

func buildCRWatcher(cfg *restclient.Config) (*crwatcher.CRWatcher, error) {
	cwCfg := &crwatcher.Config{
		PluralName: viper.GetString("crd.name"),
		Group:      viper.GetString("crd.group"),
		Version:    viper.GetString("crd.version"),
		Namespace:  viper.GetString("crd.namespace"),
		Filter:     viper.GetString("crd.filter"),
	}
	ctlr := getController()
	l := &crLogger{logger: logger}
	return crwatcher.NewCRWatcher(cwCfg, cfg, ctlr, l)
}

func getController() crwatcher.ResourceController {
	if viper.GetBool("nop") {
		logger = logger.With("controller", "print")
		logger.Info("nop specified, using the print controller")
		return &printctlr.Controller{}
	}
	if viper.GetString("helm.chart") != "" {

		chrt := viper.GetString("helm.chart")
		hns := viper.GetString("helm.namespace")
		hrn := viper.GetString("helm.releasePrefix")
		ht := viper.GetString("helm.tiller")
		hw := viper.GetBool("helm.wait")
		hwto := viper.GetInt64("helm.waitTimeout")
		logger = logger.With("controller", "helm")
		logger.Infow("using helm controller for deployment",
			"helmChart", chrt,
			"helmNamespace", hns,
			"helmReleasePrefix", hrn,
			"helmTiller", ht,
			"helmWait", hw,
			"helmWaitTimeout", hwto,
		)
		return helmctlr.NewController(chrt, hns, hrn, ht, hw, hwto, logger)
	}
	logger = logger.With("controller", "template")
	logger.Infow("using template controller for deployment", "templateDir", viper.GetString("templates"))
	viper.SetDefault("batchMode", false)
	return tmplctlr.NewController(viper.GetString("templates"), viper.GetString("k8s.config"), logger, viper.GetBool("batchMode"))
}

type crLogger struct {
	logger *zap.SugaredLogger
}

func (c crLogger) Error(err error) {
	c.logger.Errorw("kubernetes error", "error", err)
}

func validateOptions() error {
	if viper.GetString("crd.name") != "" && viper.GetString("cm.type") != "" {
		return errors.New("crd-name and cm-type cannot both be set")
	}
	if viper.GetString("crd.name") == "" && viper.GetString("cm.type") == "" {
		return errors.New("either crd-name or cm-type must be set")
	}
	if viper.GetString("crd.name") != "" {
		return validateCRDOptions()
	}
	if viper.GetString("cm.type") != "" {
		return validateCMOptions()
	}

	return nil
}

func validateCRDOptions() error {
	if viper.GetString("crd.group") == "" {
		return errors.New("crd-group is a required parameter")
	}
	if viper.GetString("crd.version") == "" {
		return errors.New("crd-version is a required parameter")
	}
	return nil
}

func validateCMOptions() error {
	if viper.GetString("cm.namespace") == "" && viper.GetBool("cm.allNamespaces") == false {
		return errors.New("either cm-namespace or cm-all-namespaces must be set")
	}
	if viper.GetString("cm.namespace") != "" && viper.GetBool("cm.allNamespaces") == true {
		return errors.New("cm-namespace and cm-all-namespaces cannot both be set")
	}
	return nil
}

type Watcher interface {
	Watch(stopCh <-chan struct{}) error
}

func startServer() error {
	if err := validateOptions(); err != nil {
		return err
	}

	version.Print(logger)

	cfg, err := getKubeClient()
	if err != nil {
		return err
	}

	var crw Watcher
	if viper.GetString("crd.name") != "" {
		crw, err = buildCRWatcher(cfg)
		if err != nil {
			return err
		}
	} else if viper.GetString("cm.type") != "" {
		crw, err = buildCMShim(cfg)
		if err != nil {
			return err
		}
	}

	// Set up Prometheus and Status endpoints.
	http.Handle(viper.GetString("server.metricsEndpoint"), promhttp.Handler())
	http.HandleFunc(viper.GetString("server.statusEndpoint"), status.Handler)
	go func() {
		err := http.ListenAndServe(viper.GetString("server.address"), nil)
		if err != nil {
			panic(err.Error())
		}
	}()

	return crw.Watch(wait.NeverStop)
}
