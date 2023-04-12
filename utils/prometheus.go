package utils

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"

	autoscaler "buildpiper.opstreelabs.in/autoscaler/api/v1"
	v1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	main "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type PrometheusParams struct {
	Name                      string
	Namespace                 string
	SVCMonitorSelector        map[string]string
	SAName                    string
	Memory                    string
	AlertManager              string
	AlertPort                 string
	Replicas                  int32
	Shards                    int32
	LogLevel                  string
	RoutePrefix               string
	Retention                 string
	DisableCompaction         bool
	ScrapeInterval            string
	ListenLocal               bool
	EnableAdminAPI            bool
	Image                     string
	EnableRemoteWriteReceiver bool
	ExternalUrl               string
	RetentionSize             string
	Paused                    bool
	OutOfOrderTimeWindow      string
	WalCompression            bool
	LogFormat                 string
	RemoteWriteDashboards     bool
	AdditionalScrapeConfigs   *main.SecretKeySelector
	IgnoreNamespaceSelectors  bool
	QueryLogFile              string
}

func GetPrometheusInstance(cr *autoscaler.CustomAutoScaling) (*v1.Prometheus, error) {
	logger := k8sLogger(cr.Namespace, cr.Name+"-prometheus-instance")
	client, err := generatePromClient()

	if err != nil {
		logger.Error(fmt.Errorf("error while fetching prometheus client  %s  in namespace %s : %s", cr.Name, cr.Namespace, err.Error()), "")
		panic(err)
	}

	promInstance, err := client.MonitoringV1().Prometheuses(cr.Namespace).Get(context.TODO(), cr.Name+"-prometheus-instance", metav1.GetOptions{})
	if err != nil {
		logger.Error(fmt.Errorf("unable to create clusterrolebinding %s", err.Error()), "")
		return nil, err
	}

	logger.Info("prometheus instance fetched succesfully")
	return promInstance, err
}

// Create a new Prometheus instance.
func CreatePrometheusInstance(cr *autoscaler.CustomAutoScaling) (*v1.Prometheus, error) {
	logger := k8sLogger(cr.Namespace, cr.Name+"-prometheus-instance")
	client, err := generatePromClient()

	if err != nil {
		logger.Error(fmt.Errorf("error while fetching prometheus client  %s  in namespace %s : %s", cr.Name, cr.Namespace, err.Error()), "")
		panic(err)
	}
	promData := PrometheusParams{
		Name:      cr.Name + "-prometheus-instance",
		Namespace: cr.Namespace,
		SVCMonitorSelector: map[string]string{
			"team": "frontend",
		},
		Image:             "quay.io/prometheus/prometheus:v2.42.0",
		SAName:            cr.Name + "-sa",
		Memory:            cr.Spec.ScalingParamsMapping["memory"],
		AlertManager:      cr.Name + "-alert",
		AlertPort:         "alert-port",
		Replicas:          3,
		Shards:            1,
		LogLevel:          "info",
		RoutePrefix:       "/",
		Retention:         "20d",
		DisableCompaction: false,
		ScrapeInterval:    "30s",
		ListenLocal:       false,
		EnableAdminAPI:    false,
		// since we will use external data sources to send the metrics
		EnableRemoteWriteReceiver: true,
		ExternalUrl:               "",
		RetentionSize:             "",
		Paused:                    false,
		OutOfOrderTimeWindow:      "0s",
		WalCompression:            true,
		LogFormat:                 "logfmt",
		RemoteWriteDashboards:     true,
		IgnoreNamespaceSelectors:  false,
		QueryLogFile:              "",
		// need to define the secret
		AdditionalScrapeConfigs: &main.SecretKeySelector{
			LocalObjectReference: main.LocalObjectReference{
				Name: cr.Name + "-secret",
			},

			Key: "scrape-config.yml",
		},
	}

	promDef, err := generatePrometheusDef(promData, cr)
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

func generatePrometheusDef(params PrometheusParams, cr *autoscaler.CustomAutoScaling) (*v1.Prometheus, error) {
	secretName := cr.Name + "-secret"
	logger := k8sLogger(cr.Namespace, secretName)

	var err error

	_, err = getSecret(cr, secretName)

	if err != nil {
		if errors.IsAlreadyExists(err) || errors.IsNotFound(err) {
			logger.Info("Secret doesnt exist for scrape config , creating now .......")
			_, err := createSecret(cr)
			if err != nil {
				return nil, err
			}

		} else {
			logger.Error(fmt.Errorf("get secret failed %s  : %s", cr.Name+"-secret", err.Error()), "")
			return nil, err
		}

	}
	lbls := generatePromLabels(params.Name, cr.Spec.ApplicationRef.DeploymentName, cr.Labels)
	objectMeta := generateObjectMetaInformation(params.Name, cr.Namespace, lbls, cr.Annotations)

	prometheus := &v1.Prometheus{
		TypeMeta: generateMetaInformation("Prometheus", "monitoring.coreos.com/v1"),

		ObjectMeta: objectMeta,

		Spec: v1.PrometheusSpec{
			// depricated
			BaseImage: params.Image,

			Alerting: &v1.AlertingSpec{
				Alertmanagers: []v1.AlertmanagerEndpoints{

					{

						Namespace: params.Namespace,
						Name:      params.AlertManager,
						Port: intstr.IntOrString{
							Type:   intstr.String,
							StrVal: params.AlertPort,
						},
					},
				},
			},

			CommonPrometheusFields: v1.CommonPrometheusFields{
				// this has more precedence
				Image: &params.Image,
				ServiceMonitorSelector: &metav1.LabelSelector{
					MatchLabels: params.SVCMonitorSelector,
				},
				Replicas: &params.Replicas,

				Resources: main.ResourceRequirements{
					Requests: map[main.ResourceName]resource.Quantity{
						main.ResourceMemory: resource.MustParse(params.Memory),
					},
				},
				// LogLevel:                  params.LogLevel,
				// LogFormat:                 params.LogFormat,
				ScrapeInterval: v1.Duration(params.ScrapeInterval),
				// EnableRemoteWriteReceiver: params.EnableRemoteWriteReceiver,
				// RoutePrefix:               params.RoutePrefix,
				// ListenLocal:               params.ListenLocal,
				// AdditionalScrapeConfigs:   params.AdditionalScrapeConfigs,
				// IgnoreNamespaceSelectors:  params.IgnoreNamespaceSelectors,
			},
			Retention:         v1.Duration(params.Retention),
			RetentionSize:     v1.ByteSize(params.RetentionSize),
			DisableCompaction: params.DisableCompaction,
			QueryLogFile:      params.QueryLogFile,

			EnableAdminAPI: true,
		},
	}

	return prometheus, nil

}

func CreatePrometheusService(cr *autoscaler.CustomAutoScaling) (*main.Service, error) {
	name := cr.Name + "-prometheus-service"
	logger := k8sLogger(cr.Namespace, name)

	params := ServiceParams{
		Name:       name,
		Namespace:  cr.Namespace,
		Port:       9090,
		TargetPort: 9090,
		TargetApp:  cr.Name + "-prometheus-instance",
		Type:       "NodePort",
		NodePort:   30901,
	}

	service, err := CreateService(cr, params)

	if err != nil {

		logger.Error(fmt.Errorf("error while creating prometheus  service  %s  in namespace %s : %s", name, cr.Namespace, err.Error()), "")
		panic(err)
	}

	logger.Info("Prometheus service created succesfully")

	return service, nil

}
