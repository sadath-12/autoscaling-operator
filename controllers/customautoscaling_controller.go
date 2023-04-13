package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	autoscaler "buildpiper.opstreelabs.in/autoscaler/api/v1"
	utils "buildpiper.opstreelabs.in/autoscaler/utils"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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

	// retrieve the cr
	instance := &autoscaler.CustomAutoScaling{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, instance)

	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		reqLogger.Error(fmt.Errorf("error while fetching CustomAutoscaling  %s", err.Error()), "")
		return ctrl.Result{}, nil
	}

	if _, found := instance.ObjectMeta.GetAnnotations()["buildpiper.opstreelabs.in/skip-reconcile"]; found {
		reqLogger.Info("Found annotations buildpiper.opstreelabs.in/skip-reconcile", "so skipping reconcile")
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}

	// handler finalizer

	if err := utils.HandleAutoScalerFinalizer(instance, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	if err := utils.AddCustomautoscaleFinalizer(instance, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	// check sa

	_, err = utils.GetSAccount(instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("service account", instance.Name+"-sa", "doesnt exists creating service account ...............")
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

	// // check role

	_, err = utils.GetClusterRole(instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("Cluster Role", instance.Name+"--clusterrole", "doesnt exists creating now .....")
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

	// // check rolebinding

	_, err = utils.GetRoleBinding(instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("Cluster Rolebindng", instance.Name+"-rolebinding", " doesnt exists creating now .....")
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
			reqLogger.Info("ServiceMonitor", instance.Name+"-svcm", "doesnt exist creating now ....")
			_, err = utils.CreateSVCMonitor(instance)
			if err != nil {
				reqLogger.Error(fmt.Errorf("error while creating service monitor for prometheus %s", err.Error()), "")
				return ctrl.Result{}, nil
			}
		} else {
			reqLogger.Error(fmt.Errorf("error while fetching service monitor for prometheus %s", err.Error()), "")
		}
	}

	// create alert managers with config and rules

	_, err = utils.GetAlertManager(instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("AlertManager", instance.Name+"-alert", "doesnt exist creating now ....")

			_, err = utils.CreateAlertManager(instance, 3)
			if err != nil {
				reqLogger.Error(fmt.Errorf("error while creating alertmanager  %s", err.Error()), "")
				return ctrl.Result{}, nil
			}
		} else {
			reqLogger.Error(fmt.Errorf("error while fetching alertmanager for prometheus %s", err.Error()), "")
		}
	}

	_, err = utils.GetPrometheusRule(instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info(instance.Name, "-prometheus-rule", "doesnt exist creating now ....")
			_, err = utils.CreatePrometheusRule(instance)
			if err != nil {
				reqLogger.Error(fmt.Errorf("error while creating prometheus rule  %s", err.Error()), "")
				return ctrl.Result{RequeueAfter: time.Second * 15}, nil
			}
		} else {
			reqLogger.Error(fmt.Errorf("error while fetching service monitor for prometheus %s", err.Error()), "")
		}
	}

	_, err = utils.GetPrometheusInstance(instance)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info(instance.Name, "-prometheus-instance", "doesnt exist creating now ....")
			_, err = utils.CreatePrometheusInstance(instance)
			if err != nil {
				reqLogger.Error(fmt.Errorf("error while creating prometheus instance  %s", err.Error()), "")
				return ctrl.Result{}, nil
			}
		} else {
			reqLogger.Error(fmt.Errorf("error while fetching service monitor for prometheus %s", err.Error()), "")
		}
	}

	// add prometheus Rule

	// find and scale the deployment

	return ctrl.Result{RequeueAfter: time.Second * 10}, nil
}

func (r *CustomAutoScalingReconciler) SetupWebhookServer(mgr manager.Manager) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", r.handleWebhook)

	server := &http.Server{
		Addr:    ":3030",
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CustomAutoScalingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.SetupWebhookServer(mgr); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&autoscaler.CustomAutoScaling{}).
		Complete(r)
}

func (r *CustomAutoScalingReconciler) handleWebhook(w http.ResponseWriter, req *http.Request) {

	instance := &autoscaler.CustomAutoScaling{}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var alert utils.AlertmanagerPayload
	if err := json.Unmarshal(body, &alert); err != nil {
		http.Error(w, "Failed to unmarshal alert payload", http.StatusBadRequest)
		return
	}

	// Extract relevant information from alert, such as alert name and severity
	// alertName := alert.Alerts[0].Labels["alertname"]
	alertSeverity := alert.Alerts[0].Labels["severity"]

	// Determine desired number of replicas based on alert information
	var desiredReplicas int32
	desiredReplicas = 0
	if alertSeverity == "critical" {
		desiredReplicas = 5
	} else if alertSeverity == "warning" {
		desiredReplicas = 3
	} else {
		desiredReplicas = 1
	}

	// Update deployment replica count using Kubernetes API client
	deployment := &appsv1.Deployment{}

	if err := r.Get(context.Background(), types.NamespacedName{Name: instance.Spec.ApplicationRef.DeploymentName, Namespace: instance.Namespace}, deployment); err != nil {
		http.Error(w, "Failed to retrieve deployment", http.StatusInternalServerError)
		return
	}

	deployment.Spec.Replicas = &desiredReplicas
	if err := r.Update(context.Background(), deployment); err != nil {
		http.Error(w, "Failed to update deployment", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
}
