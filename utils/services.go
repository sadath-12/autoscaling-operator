package utils

import (
	"context"
	"fmt"

	autoscaler "buildpiper.opstreelabs.in/autoscaler/api/v1"
	main "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("controller_autoscaler")

type ServiceParams struct {
	Name       string
	Namespace  string
	Port       int
	PortName   string
	TargetPort int
	TargetApp  string
	Type       string
	NodePort   int
}

func GetService(cr *autoscaler.CustomAutoScaling, name string) (*main.Service, error) {

	logger := k8sLogger(cr.Namespace, name)
	service, err := generateK8sClient().CoreV1().Services(cr.Namespace).Get(context.TODO(), name, metav1.GetOptions{})

	if err != nil {

		logger.Error(fmt.Errorf("error while fetching   service  %s  in namespace %s : %s", name, cr.Namespace, err.Error()), "")
		return nil, err
	}

	logger.Info(name + "service fetched succesfully")

	return service, nil

}

func CreateService(cr *autoscaler.CustomAutoScaling, params ServiceParams) (*main.Service, error) {

	logger := k8sLogger(cr.Namespace, params.Name)

	serviceDef := generateServiceDef(cr, params)
	service, err := generateK8sClient().CoreV1().Services(cr.Namespace).Create(context.TODO(), serviceDef, metav1.CreateOptions{})

	if err != nil {

		logger.Error(fmt.Errorf("error while creating   service  %s  in namespace %s : %s", params.Name, cr.Namespace, err.Error()), "")
		panic(err)
	}

	logger.Info(params.Name + "service created succesfully")

	return service, nil

}

func generateServiceDef(cr *autoscaler.CustomAutoScaling, params ServiceParams) *main.Service {

	service := &main.Service{
		TypeMeta:   generateMetaInformation("Service", "v1"),
		ObjectMeta: generateObjectMetaInformation(params.Name, cr.Namespace, cr.Labels, cr.Annotations),
		Spec: main.ServiceSpec{
			Selector: map[string]string{
				"app": params.TargetApp,
			},
			Ports: []main.ServicePort{
				{
					Name:       params.PortName,
					Port:       int32(params.Port),
					TargetPort: intstr.FromInt(params.TargetPort),
					NodePort:   int32(params.NodePort),
				},
			},
			Type: main.ServiceType(params.Type),
		},
	}

	return service

}
