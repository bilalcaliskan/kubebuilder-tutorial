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

/*
Our package starts out with some basic imports. Particularly:
  - The core controller-runtime library
  - The default controller-runtime logging, Zap (more on that a bit later)

What does controller-runtime library?
The Kubernetes controller-runtime Project is a set of go libraries for building Controllers. It is leveraged by
Kubebuilder and Operator SDK.

Package controllerruntime provides tools to construct Kubernetes-style controllers that manipulate both Kubernetes
CRDs and aggregated/built-in Kubernetes APIs.

It defines easy helpers for the common use cases when building CRDs, built on top of customizable layers of abstraction.
Common cases should be easy, and uncommon cases should be possible. In general, controller-runtime tries to guide users
towards Kubernetes controller best-practices.

The main entrypoint for controller-runtime is this root package, which contains all of the common types needed to get
started building controllers:

import (
    ctrl "sigs.k8s.io/controller-runtime"
)
*/

import (
	"flag"
	batchv1 "github.com/bilalcaliskan/kubebuilder-tutorial/apis/batch/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/bilalcaliskan/kubebuilder-tutorial/controllers"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	configv1 "github.com/bilalcaliskan/kubebuilder-tutorial/apis/config/v1"
	//+kubebuilder:scaffold:imports
)

var (
	/*
		Every set of controllers needs a Scheme, which provides mappings between Kinds and their corresponding Go types.

		scheme defines methods for serializing and deserializing API objects, a type registry for converting group, version,
		and kind information to and from Go schemas, and mappings between Go schemas of different versions. A scheme is the
		foundation for a versioned API and versioned configuration over time.

		In a scheme, a Type is a particular Go struct, a Version is a point-in-time identifier for a particular
		representation of that Type (typically backwards compatible), a Kind is the unique name for that Type within the
		Version, and a Group identifies a set of Versions, Kinds, and Types that evolve over time. An Unversioned Type is
		one that is not yet formally bound to a type and is promised to be backwards compatible (effectively a "v1" of a
		Type that does not expect to break in the future).

		Schemes are not expected to change at runtime and are only threadsafe after registration is complete.
	*/
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	/*
		Notice is that kubebuilder has added the new API group’s package (batchv1) to our scheme. This means that we can
		use those objects in our controller.

		If we would be using any other CRD we would have to add their scheme the same way. Builtin types such as Job have
		their scheme added by clientgoscheme.
	*/
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(batchv1.AddToScheme(scheme))
	utilruntime.Must(configv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	/*
		At this point, our main function is fairly simple:
			- We set up some basic flags for metrics.
			- We instantiate a manager, which keeps track of running all of our controllers, as well as setting up
			shared caches and clients to the API server (notice we tell the manager about our Scheme).
			- We run our manager, which in turn runs all of our controllers and webhooks. The manager is set up to
			run until it receives a graceful shutdown signal. This way, when we’re running on Kubernetes, we behave
			nicely with graceful pod termination.

		Package manager(sigs.k8s.io/controller-runtime/pkg/manager) is required to create Controllers and provides
		shared dependencies such as clients, caches, schemes, etc. Controllers must be started by calling Manager.Start.
	*/

	// First, add a new flag to specify the path that the component config file should be loaded from.
	var configFile string
	flag.StringVar(&configFile, "config", "config/manager/controller_manager_config.yaml",
		"The controller will load its initial configuration from this file. Omit this flag to use the "+
			"default configuration values. Command-line flags override configuration from this file.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	/*
		Now, we can setup the Options struct and check if the configFile is set, this allows backwards compatibility,
		if it’s set we’ll then use the AndFrom function on Options to parse and populate the Options from the config.
	*/
	var err error
	options := ctrl.Options{Scheme: scheme}
	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile))
		if err != nil {
			setupLog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}

	// Lastly, we’ll change the NewManager call to use the options varible we defined above.
	var mgr manager.Manager
	if mgr, err = ctrl.NewManager(ctrl.GetConfigOrDie(), options); err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Kubebuilder has added a block calling our CronJob controller’s SetupWithManager method.
	if err = (&controllers.CronJobReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CronJob")
		os.Exit(1)
	}

	// +kubebuilder:docs-gen:collapse=existing setup

	// Our existing call to SetupWebhookWithManager registers our conversion webhooks with the manager, too.
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&batchv1.CronJob{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "CronJob")
			os.Exit(1)
		}
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
