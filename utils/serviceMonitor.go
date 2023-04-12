package utils

import (
	"context"
	"fmt"

	autoscaler "buildpiper.opstreelabs.in/autoscaler/api/v1"
	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SVCMonitorParams struct {
	Name       string
	ObjectMeta metav1.ObjectMeta
	Namespace  string
	selector   map[string]string
	Endpoints  []v1.Endpoint
	Image      string
}

func generateSVCMonitorDef(cr *autoscaler.CustomAutoScaling, params SVCMonitorParams) *v1.ServiceMonitor {

	lbls := generateSVCMLabels(params.Name, cr.ObjectMeta.Labels)
	svcMonitor := &v1.ServiceMonitor{
		TypeMeta: generateMetaInformation("ServiceMonitor", "monitoring.coreos.com/v1"),

		ObjectMeta: generateObjectMetaInformation(params.Name, params.Namespace, lbls, params.ObjectMeta.Annotations),
		Spec: v1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: params.selector,
			},

			Endpoints: params.Endpoints,
		},
	}

	return svcMonitor

}

func GetSVCMonitor(cr *autoscaler.CustomAutoScaling) (*v1.ServiceMonitor, error) {
	svcMonitorName := cr.Name + "-svcm"
	logger := k8sLogger(cr.Namespace, svcMonitorName)
	client, err := generatePromClient()

	if err != nil {
		logger.Error(fmt.Errorf("error while fetching prometheus client  %s  in namespace %s : %s", svcMonitorName, cr.Namespace, err.Error()), "")
		panic(err)
	}

	svcMonitor, err := client.MonitoringV1().ServiceMonitors(cr.Namespace).Get(context.TODO(), svcMonitorName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logger.Info("servicemonitor doesnt exists")
			return nil, err
		}
		logger.Error(fmt.Errorf("error while fetching servicemonitor %s  in namespace %s : %s", svcMonitorName, cr.Namespace, err.Error()), "")
		return nil, err
	}

	logger.Info("ServiceMonitor fetched succesfully")

	return svcMonitor, nil
}

func CreateSVCMonitor(cr *autoscaler.CustomAutoScaling) (*v1.ServiceMonitor, error) {
	svcMonitorName := cr.Name + "-svcm"
	logger := k8sLogger(cr.Namespace, svcMonitorName)
	client, err := generatePromClient()

	if err != nil {
		logger.Error(fmt.Errorf("error while fetching prometheus client  %s  in namespace %s : %s", svcMonitorName, cr.Namespace, err.Error()), "")
		panic(err)
	}

	endpoints := []v1.Endpoint{
		{
			Port:     "metrics",
			Interval: "30s",
			Path:     "/metrics",
		},
	}

	params := SVCMonitorParams{
		Name:      svcMonitorName,
		Namespace: cr.Namespace,
		selector: map[string]string{
			"app": cr.Spec.ApplicationRef.DeploymentName,
		},
		Endpoints: endpoints,
	}
	SVCDef := generateSVCMonitorDef(cr, params)

	svcMonitor, err := client.MonitoringV1().ServiceMonitors(cr.Namespace).Create(context.TODO(), SVCDef, metav1.CreateOptions{})

	if err != nil {
		logger.Error(fmt.Errorf("error while creating servicemonitor %s  in namespace %s : %s", svcMonitorName, cr.Namespace, err.Error()), "")
		return nil, err
	}

	logger.Info("ServiceMonitor created succesfully")

	return svcMonitor, nil
}
