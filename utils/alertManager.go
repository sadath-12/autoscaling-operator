package utils

import (
	"context"
	"fmt"

	autoscaler "buildpiper.opstreelabs.in/autoscaler/api/v1"
	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	main "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AlertManagerParams struct {
	Name           string
	Namespace      string
	TypeMeta       metav1.TypeMeta
	ObjectMeta     metav1.ObjectMeta
	Replicas       int32
	ConfigSelector map[string]string
	image          string
}

func GetAlertManager(name, namespace string) (*v1.Alertmanager, error) {
	alertManagerName := name + "-alert"
	logger := k8sLogger(namespace, name+"-alert")
	client, err := generatePromClient()

	if err != nil {
		logger.Error(fmt.Errorf("error while fetching prometheus client  %s  in namespace %s : %s", name, namespace, err.Error()), "")
		panic(err)
	}

	alertManager, err := client.MonitoringV1().Alertmanagers(namespace).Get(context.TODO(), alertManagerName, metav1.GetOptions{})

	if err != nil {
		logger.Error(fmt.Errorf("unable to fetch alertManager %s", err.Error()), "")
		return nil, err
	}

	logger.Info("alert Manager fetched succesfully")
	return alertManager, err

}

func CreateAlertManager(cr *autoscaler.CustomAutoScaling, config string, replicas int32) (*v1.Alertmanager, error) {
	alertManagerName := cr.Name + "-alert"
	logger := k8sLogger(cr.Namespace, cr.Name+"-alert")
	client, err := generatePromClient()

	if err != nil {
		logger.Error(fmt.Errorf("error while fetching prometheus client  %s  in namespace %s : %s", alertManagerName, cr.Namespace, err.Error()), "")
		panic(err)
	}

	labels := generateAlertLabels(alertManagerName, "Cluster", cr.ObjectMeta.Labels)
	annotations := generateAlertAnots(cr.ObjectMeta)

	params := AlertManagerParams{
		Name:       alertManagerName,
		Namespace:  cr.Namespace,
		TypeMeta:   generateMetaInformation("Alertmanager", "monitoring.coreos.com/v1"),
		ObjectMeta: generateObjectMetaInformation(alertManagerName, cr.Namespace, labels, annotations),
		ConfigSelector: map[string]string{
			"name": config,
		},
		Replicas: replicas,
		image:    "prometheus/alertmanager:v0.25.0",
	}

	alertManagerDef := generateAlertManagerDef(params)

	alertManager, err := client.MonitoringV1().Alertmanagers(params.Namespace).Create(context.TODO(), alertManagerDef, metav1.CreateOptions{})

	if err != nil {
		logger.Error(fmt.Errorf("unable to create alertManager %s", err.Error()), "")
		return nil, err
	}

	logger.Info("alert Manager created succesfully")

	return alertManager, nil

}

func generateAlertManagerDef(params AlertManagerParams) *v1.Alertmanager {

	runAsGroup := int64(2000)
	runAsUser := int64(1000)
	fsGroup := int64(2000)
	runAsNonRoot := bool(true)
	clusterMode := bool(false)

	alertManager := v1.Alertmanager{
		TypeMeta: generateMetaInformation("Alertmanager", "monitoring.coreos.com/v1"),
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.Name,
			Namespace: params.Namespace,
		},

		Spec: v1.AlertmanagerSpec{
			Replicas: &params.Replicas,

			AlertmanagerConfigSelector: &metav1.LabelSelector{
				MatchLabels: params.ConfigSelector,
			},
			Image: &params.image,
			SecurityContext: &main.PodSecurityContext{
				RunAsUser:    &runAsUser,
				RunAsNonRoot: &runAsNonRoot,
				FSGroup:      &fsGroup,
				RunAsGroup:   &runAsGroup,
			},
			ClusterAdvertiseAddress: "",
			ForceEnableClusterMode:  clusterMode,
		},
	}

	return &alertManager
}
