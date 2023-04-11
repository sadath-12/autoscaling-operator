package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	autoscaler "buildpiper.opstreelabs.in/autoscaler/api/v1"
	utils "buildpiper.opstreelabs.in/autoscaler/utils"
)

// CustomAutoScalingReconciler reconciles a CustomAutoScaling object
type CustomAutoScalingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var log = logf.Log.WithName("controller_autoscaler")

//+kubebuilder:rbac:groups=buildpiper.opstreelabs.in,resources=customautoscalings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=buildpiper.opstreelabs.in,resources=customautoscalings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=buildpiper.opstreelabs.in,resources=customautoscalings/finalizers,verbs=update

func (r *CustomAutoScalingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("Request.Service.Namespace", "Request.Service.Name", req.Namespace, req.Name)

	reqLogger.Info("reconcilling autoscaler.....")

	var memory string

	// retrieve the cr
	instance := &autoscaler.CustomAutoScaling{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)

	memory = instance.Spec.ScalingParamsMapping["memory"]

	if memory == "" {
		memory = "256Mi"
	}

	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		reqLogger.Error(fmt.Errorf("error while fetching CustomAutoscaling  %s", err.Error()), "")
		return ctrl.Result{}, nil

	}

	// handler finalizer

	// check sa

	_, err = utils.GetSAccount(instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("service account %s doesnt exists creating service account ...............", instance.Name+"-sa")
			_, err = utils.CreateServiceAccount(instance)
			if err != nil {
				fmt.Print("getting error while creating acc .............. ", err.Error())
				reqLogger.Error(fmt.Errorf("error while creating service account for prometheus %s", err.Error()), "")
				return ctrl.Result{}, nil
			}
		} else {
			reqLogger.Info("Cant fetch servicemonitor")
			return ctrl.Result{}, nil
		}
	}

	// check role

	_, err = utils.GetClusterRole(instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("Cluster Role %s doesnt exists creating now .....", instance.Name+"--clusterrole")
			_, err = utils.CreateClusterRole(instance)
			if err != nil {
				reqLogger.Error(fmt.Errorf("error while creating cluster role for prometheus %s", err.Error()), "")
				return ctrl.Result{}, nil
			}
		} else {
			reqLogger.Info("Cant fetch clusterrole")
			return ctrl.Result{}, nil
		}
	}

	// check rolebinding

	_, err = utils.GetRoleBinding(instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("Cluster Rolebindng %s doesnt exists creating now .....", instance.Name+"-rolebinding")
			_, err = utils.CreateClusterRoleBinding(instance)
			if err != nil {
				reqLogger.Error(fmt.Errorf("error while creating cluster role for prometheus %s", err.Error()), "")
				return ctrl.Result{}, nil
			}
		} else {
			reqLogger.Info("Cant fetch clusterrole")
			return ctrl.Result{}, nil
		}
	}

	// if servicemonitor is not der then create monitoring that deployment
	_, err = utils.GetSVCMonitor(instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("ServiceMonitor %s doesnt exist creating now ....", instance.Name+"-svcm")
			_, err = utils.CreateSVCMonitor(instance)
			if err != nil {
				reqLogger.Error(fmt.Errorf("error while creating service monitor for prometheus %s", err.Error()), "")
				return ctrl.Result{}, nil
			}
		} else {
			reqLogger.Error(fmt.Errorf("error while fetching service monitor for prometheus %s", err.Error()), "")
		}
	}

	// create alertManager config

	// create alert managers

	// create rules

	// if prometheus not der then create prometheus instance

	_, err = utils.GetPrometheusInstance(instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("Prometheus Instance %s doesnt exist creating now ....", instance.Name+"-instance")
			_, err = utils.CreatePrometheusInstance(instance)
			if err != nil {
				reqLogger.Error(fmt.Errorf("error while creating prometheus instance  %s", err.Error()), "")
				return ctrl.Result{}, nil
			}
		} else {
			reqLogger.Error(fmt.Errorf("error while fetching service monitor for prometheus %s", err.Error()), "")
		}

	}

	// create prometheus service

	// retreive the alert from the manager through webhook if firing

	// calculate the replica

	// find and scale the deployment

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CustomAutoScalingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&autoscaler.CustomAutoScaling{}).
		Complete(r)
}
