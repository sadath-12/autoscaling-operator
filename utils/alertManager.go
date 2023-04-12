package utils

import (
	"context"

	"fmt"

	autoscaler "buildpiper.opstreelabs.in/autoscaler/api/v1"
	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	main "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	Secrets        []string
}

type AlertmanagerPayload struct {
	Receiver          string            `json:"receiver"`
	Status            string            `json:"status"`
	Alerts            []Alert           `json:"alerts"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
}

type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     string            `json:"startsAt"`
	EndsAt       string            `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

func GetAlertManager(cr *autoscaler.CustomAutoScaling) (*v1.Alertmanager, error) {
	alertManagerName := cr.Name + "-alert"
	logger := k8sLogger(cr.Namespace, cr.Name+"-alert")
	client, err := generatePromClient()

	if err != nil {
		logger.Error(fmt.Errorf("error while fetching prometheus client  %s  in namespace %s : %s", cr.Name, cr.Namespace, err.Error()), "")
		panic(err)
	}

	alertManager, err := client.MonitoringV1().Alertmanagers(cr.Namespace).Get(context.TODO(), alertManagerName, metav1.GetOptions{})

	if err != nil {
		logger.Error(fmt.Errorf("unable to fetch alertManager %s", err.Error()), "")
		return nil, err
	}

	logger.Info("alert Manager fetched succesfully")
	return alertManager, err

}

func CreateAlertManager(cr *autoscaler.CustomAutoScaling, replicas int32) (*v1.Alertmanager, error) {
	alertManagerName := cr.Name + "-alert"
	logger := k8sLogger(cr.Namespace, cr.Name+"-alert")
	client, err := generatePromClient()

	if err != nil {
		logger.Error(fmt.Errorf("error while fetching prometheus client  %s  in namespace %s : %s", alertManagerName, cr.Namespace, err.Error()), "")
		panic(err)
	}

	_, err = getSecret(cr, alertManagerName+"secret")
	if err != nil {
		if errors.IsAlreadyExists(err) || errors.IsNotFound(err) {

			_, err = createAlertConfigSecret(cr)
			if err != nil {
				panic(err)
			}
		}
		panic(err)

	}

	if err != nil {
		logger.Error(fmt.Errorf("error while creating alert secret %s  in namespace %s : %s", alertManagerName+"secret", cr.Namespace, err.Error()), "")
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
			"name": alertManagerName + "config",
		},
		Replicas: replicas,
		image:    "quay.io/prometheus/alertmanager:v0.25.0",
		Secrets:  []string{alertManagerName + "secret"},
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
			Secrets: params.Secrets,
			// Image:   &params.image,
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

func CreateAlertManagerService(cr *autoscaler.CustomAutoScaling) (*main.Service, error) {
	name := cr.Name + "-alert-service"
	logger := k8sLogger(cr.Namespace, name)

	params := ServiceParams{
		Name:       name,
		Namespace:  cr.Namespace,
		Port:       9093,
		TargetPort: 9093,
		TargetApp:  cr.Name + "-alert",
		Type:       "NodePort",
		NodePort:   30900,
	}

	service, err := CreateService(cr, params)

	if err != nil {

		logger.Error(fmt.Errorf("error while creating prometheus  service  %s  in namespace %s : %s", name, cr.Namespace, err.Error()), "")
		panic(err)
	}

	logger.Info("Prometheus service created succesfully")

	return service, nil

}
