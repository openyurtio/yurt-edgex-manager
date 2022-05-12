/*
Copyright 2021.

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

package main

import (
	"embed"
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	unitv1alpha1 "github.com/openyurtio/yurt-app-manager-api/pkg/yurtappmanager/apis/apps/v1alpha1"
	devicev1alpha1 "github.com/openyurtio/yurt-edgex-manager/api/v1alpha1"
	"github.com/openyurtio/yurt-edgex-manager/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"
	//+kubebuilder:scaffold:imports
)

type EdgeXConfiguration struct {
	Version     string                                  `json:"version,omitempty"`
	Configmap   corev1.ConfigMap                        `json:"configmap,omitempty"`
	Services    []devicev1alpha1.ServiceTemplateSpec    `json:"services,omitempty"`
	Deployments []devicev1alpha1.DeploymentTemplateSpec `json:"deployments,omitempty"`
}

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
	//go:embed EdgeXConfig
	edgeXconfig embed.FS
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(devicev1alpha1.AddToScheme(scheme))

	utilruntime.Must(unitv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	files, err := edgeXconfig.ReadDir("EdgeXConfig")
	if err != nil {
		setupLog.Error(err, "File to open the embed EdgeX config")
		os.Exit(1)
	}
	for _, file := range files {
		var edgexconfig EdgeXConfiguration
		if file.IsDir() {
			continue
		}
		configdata, err := edgeXconfig.ReadFile("EdgeXConfig/" + file.Name())
		if err != nil {
			setupLog.Error(err, "File to open the embed EdgeX config")
			os.Exit(1)
		}
		err = yaml.Unmarshal(configdata, &edgexconfig)
		if err != nil {
			setupLog.Error(err, "Wrong edgeX configuration file")
			os.Exit(1)
		}
		controllers.CoreDeployment[edgexconfig.Version] = edgexconfig.Deployments
		controllers.CoreServices[edgexconfig.Version] = edgexconfig.Services
		controllers.CoreConfigMap[edgexconfig.Version] = edgexconfig.Configmap
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "31095ea9.openyurt.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.EdgeXReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "EdgeX")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
