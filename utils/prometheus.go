package utils

import (
	"context"
	"fmt"

	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	main "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	autoscaler "buildpiper.opstreelabs.in/autoscaler/api/v1"
)

type PrometheusParams struct {
	Name               string
	Namespace          string 
	SVCMonitorSelector map[string]string
	SAName             string
	Memory             string
	AlertManager       string
	AlertPort          string
	RulesSelector      map[string]string
	Replicas           int32
}

type SVCMonitorParams struct {
	Name       string
	ObjectMeta metav1.ObjectMeta
	Namespace  string
	selector   map[string]string
	Endpoints  []v1.Endpoint
}

func GetPrometheusInstance(cr *autoscaler.CustomAutoScaling) (*v1.Prometheus, error) {
	logger := k8sLogger(cr.Namespace, cr.Name+"-instance")
	client, err := generatePromClient()

	if err != nil {
		logger.Error(fmt.Errorf("error while fetching prometheus client  %s  in namespace %s : %s", cr.Name, cr.Namespace, err.Error()), "")
		panic(err)
	}

	promInstance, err := client.MonitoringV1().Prometheuses(cr.Namespace).Get(context.TODO(), cr.Name+"-instance", metav1.GetOptions{})
	if err != nil {
		logger.Error(fmt.Errorf("unable to create clusterrolebinding %s", err.Error()), "")
		return nil, err
	}

	logger.Info("prometheus instance fetched succesfully")
	return promInstance, err
}

// Create a new Prometheus instance.
func CreatePrometheusInstance(cr *autoscaler.CustomAutoScaling ) (*v1.Prometheus, error) {
	logger := k8sLogger(cr.Namespace, cr.Name+"-instance")
	client, err := generatePromClient()

	if err != nil {
		logger.Error(fmt.Errorf("error while fetching prometheus client  %s  in namespace %s : %s", cr.Name, cr.Namespace, err.Error()), "")
		panic(err)
	}
	promData := PrometheusParams{
		Name:     cr.Name + "-instance",
		Namespace: cr.Namespace,
		SVCMonitorSelector: map[string]string{
			"app": cr.Name+"svcm",
		},
		SAName: cr.Name+"-sa",
		Memory: cr.Spec.ScalingParamsMapping["memory"],
	}
	promDef, err := generatePrometheusDef(promData)
	if err != nil {

		logger.Error(fmt.Errorf("error while creating prometheus instance params  %s  in namespace %s : %s", cr.Name, cr.Namespace, err.Error()), "")
		panic(err)
	}

	promInstance, err := client.MonitoringV1().Prometheuses(cr.Namespace).Create(context.TODO(), promDef, metav1.CreateOptions{})

	if err != nil {
		logger.Error(fmt.Errorf("error while creating prometheus instance  %s  in namespace %s : %s", cr.Name, cr.Namespace, err.Error()), "")
	}

	logger.Info("prometheus instance created succesfully")

	return promInstance, nil
}

func generatePrometheusDef(params PrometheusParams) (*v1.Prometheus, error) {

	prometheus := &v1.Prometheus{
		TypeMeta: generateMetaInformation("Prometheus", "monitoring.coreos.com/v1"),
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.Name,
			Namespace: params.Namespace,
		},

		Spec: v1.PrometheusSpec{

			Alerting: &v1.AlertingSpec{
				Alertmanagers: []v1.AlertmanagerEndpoints{

					{

						Namespace: params.Namespace,
						Name:      params.AlertManager + "Alert",
						Port: intstr.IntOrString{
							Type:   intstr.String,
							StrVal: params.AlertPort,
						},
					},
				},
			},

			CommonPrometheusFields: v1.CommonPrometheusFields{

				ServiceMonitorSelector: &metav1.LabelSelector{
					MatchLabels: params.SVCMonitorSelector,
				},
				Replicas: &params.Replicas,

				Resources: main.ResourceRequirements{
					Requests: map[main.ResourceName]resource.Quantity{
						main.ResourceMemory: resource.MustParse(params.Memory),
					},
				},
			},
			EnableAdminAPI: false,
		},
	}

	return prometheus, nil

}
