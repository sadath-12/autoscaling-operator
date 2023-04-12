package utils

import (
	"context"

	autoscaler "buildpiper.opstreelabs.in/autoscaler/api/v1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	AutoscaleFinalizer string = "customautoscalingFinalizer"
)

func HandleAutoScalerFinalizer(cr *autoscaler.CustomAutoScaling, cl client.Client) error {
	logger := finalizerLogger(cr.Namespace, AutoscaleFinalizer)
	if cr.GetDeletionTimestamp() != nil {
		if controllerutil.ContainsFinalizer(cr, AutoscaleFinalizer) {
			if err := finalizePrometheus(cr); err != nil {
				return err
			}

			if err := finalizeAlertManager(cr); err != nil {
				return err
			}
			if err := finalizeServiceAccount(cr); err != nil {
				return err
			}
			if err := finalizeRoles(cr); err != nil {
				return err
			}
			if err := finalizeSVCMonitor(cr); err != nil {
				return err
			}

			controllerutil.RemoveFinalizer(cr, AutoscaleFinalizer)
			if err := cl.Update(context.TODO(), cr); err != nil {
				logger.Error(err, "could not remove finalizer"+AutoscaleFinalizer)
				return err
			}
		}
	}
	logger.Info("Finalized the stack succesfully")
	return nil
}

func AddCustomautoscaleFinalizer(cr *autoscaler.CustomAutoScaling, cl client.Client) error {
	if !controllerutil.ContainsFinalizer(cr, AutoscaleFinalizer) {
		controllerutil.AddFinalizer(cr, AutoscaleFinalizer)
		return cl.Update(context.TODO(), cr)
	}
	return nil

}

func finalizePrometheus(cr *autoscaler.CustomAutoScaling) error {
	logger := finalizerLogger(cr.Namespace, AutoscaleFinalizer)
	client, err := generatePromClient()

	if err != nil {
		panic(err)
	}
	promInstances := cr.Name + "-prometheus-instance"
	for _, instance := range []string{promInstances} {
		err = client.MonitoringV1().Prometheuses(cr.Namespace).Delete(context.TODO(), instance, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			logger.Error(err, "could not delete prometheus", instance)
			return err
		}
	}
	return nil
}

func finalizeAlertManager(cr *autoscaler.CustomAutoScaling) error {
	logger := finalizerLogger(cr.Namespace, AutoscaleFinalizer)
	client, err := generatePromClient()

	if err != nil {
		panic(err)
	}
	alertInstances := cr.Name + "-alert"
	for _, instance := range []string{alertInstances} {
		err = client.MonitoringV1().Alertmanagers(cr.Namespace).Delete(context.TODO(), instance, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			logger.Error(err, "could not delete alert", instance)
			return err
		}
	}
	return nil
}

func finalizeServiceAccount(cr *autoscaler.CustomAutoScaling) error {
	logger := finalizerLogger(cr.Namespace, AutoscaleFinalizer)

	saName := cr.Name + "-sa"

	err := generateK8sClient().CoreV1().ServiceAccounts(cr.Namespace).Delete(context.TODO(), saName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "could not delete service account", saName)
		return err
	}

	return nil
}

func finalizeSVCMonitor(cr *autoscaler.CustomAutoScaling) error {
	logger := finalizerLogger(cr.Namespace, AutoscaleFinalizer)
	client, err := generatePromClient()

	if err != nil {
		panic(err)
	}

	serviceMonitor := cr.Name + "-svcm"

	err = client.MonitoringV1().ServiceMonitors(cr.Namespace).Delete(context.TODO(), serviceMonitor, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "could not delete service monitor", serviceMonitor)
		return err
	}

	return nil
}

func finalizeRoles(cr *autoscaler.CustomAutoScaling) error {
	logger := finalizerLogger(cr.Namespace, AutoscaleFinalizer)

	roleName := cr.Name + "-role"

	err := generateK8sClient().RbacV1().ClusterRoles().Delete(context.TODO(), roleName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		logger.Error(err, "could not delete cluster role", roleName)
		return err
	}

	return nil
}

// finalizeLogger will generate logging interface
func finalizerLogger(namespace string, name string) logr.Logger {
	reqLogger := log.WithValues("Request.Service.Namespace", namespace, "Request.Finalizer.Name", name)
	return reqLogger
}
