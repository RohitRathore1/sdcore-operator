package main

import (
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/RohitRathore1/sdcore-operator/controllers/nf"
	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	refv1alpha1 "github.com/nephio-project/api/references/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	runscheme "sigs.k8s.io/controller-runtime/pkg/scheme"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = controllerruntime.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddress string
	var healthProbeAddress string
	var leaderElect bool

	flag.StringVar(&metricsAddress, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&healthProbeAddress, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&leaderElect, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	zapOptions := zap.Options{
		Development: true,
	}
	zapOptions.BindFlags(flag.CommandLine)
	flag.Parse()

	controllerruntime.SetLogger(zap.New(zap.UseFlagOptions(&zapOptions)))

	manager, err := controllerruntime.NewManager(controllerruntime.GetConfigOrDie(), controllerruntime.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddress,
		Port:                   9443,
		HealthProbeBindAddress: healthProbeAddress,
		LeaderElection:         leaderElect,
		LeaderElectionID:       "5089c67f.nephio.org",
	})
	if err != nil {
		fail(err, "unable to start manager")
	}

	schemeBuilder := &runscheme.Builder{GroupVersion: nephiov1alpha1.GroupVersion}

	schemeBuilder.Register(&nephiov1alpha1.NFDeployment{}, &nephiov1alpha1.NFDeploymentList{})
	if err := schemeBuilder.AddToScheme(manager.GetScheme()); err != nil {
		fail(err, "Not able to register NFDeployment kind")
	}

	schemeBuilder = &runscheme.Builder{GroupVersion: refv1alpha1.GroupVersion}
	schemeBuilder.Register(&refv1alpha1.Config{}, &refv1alpha1.ConfigList{})
	if err := schemeBuilder.AddToScheme(manager.GetScheme()); err != nil {
		fail(err, "Not able to register Config.ref kind")
	}

	if err = (&nf.NFDeploymentReconciler{
		Client: manager.GetClient(),
		Scheme: manager.GetScheme(),
	}).SetupWithManager(manager); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NFDeployment")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := manager.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		fail(err, "unable to set up health check")
	}
	if err := manager.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		fail(err, "unable to set up ready check")
	}

	setupLog.Info("starting manager")
	if err := manager.Start(controllerruntime.SetupSignalHandler()); err != nil {
		fail(err, "problem running manager")
	}
}

func fail(err error, msg string, keysAndValues ...any) {
	setupLog.Error(err, msg, keysAndValues...)
	os.Exit(1)
}
