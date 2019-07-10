package dbbroker

import (
	gallocedronev1beta1 "dbbroker/pkg/apis/gallocedrone/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

//DEPLOYMENT

// +kubebuilder:rbac:groups=apps,resources=Deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=Secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gallocedrone.gallocedrone.io,resources=dbbrokers,verbs=get;list;watch;create;update;patch;delete

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDbBroker{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("dbbroker-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to DbBroker
	err = c.Watch(&source.Kind{Type: &gallocedronev1beta1.DbBroker{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileDbBroker{}

// ReconcileDbBroker reconciles a DbBroker object
type ReconcileDbBroker struct {
	client.Client
	scheme *runtime.Scheme
}

func AddDeployment(mgr manager.Manager) error {
	return addDeployment(mgr, newReconcilerDeployment(mgr))
}

//DBBROKER

// newReconciler returns a new reconcile.Reconciler
func newReconcilerDeployment(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDbBrokerDeployment{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func addDeployment(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("dbbroker-controller-Deployment", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to DbBroker
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileDbBrokerDeployment{}

// ReconcileDbBroker reconciles a DbBroker object
type ReconcileDbBrokerDeployment struct {
	client.Client
	scheme *runtime.Scheme
}
