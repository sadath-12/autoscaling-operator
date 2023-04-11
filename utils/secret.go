package utils

import (
	"context"
	"fmt"

	autoscaler "buildpiper.opstreelabs.in/autoscaler/api/v1"
	main "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getSecret(cr *autoscaler.CustomAutoScaling) (*main.Secret, error) {
	secretName := cr.Name + "-secret"
	logger := k8sLogger(cr.Namespace, secretName)

	secret, err := generateK8sClient().CoreV1().Secrets(cr.Namespace).Get(context.TODO(), secretName, metav1.GetOptions{})

	if err != nil {

		logger.Error(fmt.Errorf("get secret  failed %s  : %s", secretName, err.Error()), "")
		return nil, err
	}

	logger.Info("Secret fetch succesfully ")

	return secret, nil
}

func createSecret(cr *autoscaler.CustomAutoScaling) (*main.Secret, error) {
	secretName := cr.Name + "-secret"
	logger := k8sLogger(cr.Namespace, secretName)

	secretDef := generateSecretDef(cr)

	secret, err := generateK8sClient().CoreV1().Secrets(cr.Namespace).Create(context.TODO(), secretDef, metav1.CreateOptions{})

	if err != nil {

		logger.Error(fmt.Errorf("create secret  failed %s  : %s", secretName, err.Error()), "")
		return nil, err
	}

	logger.Info("Secret created succesfully ")

	return secret, nil
}

func generateSecretDef(cr *autoscaler.CustomAutoScaling) *main.Secret {

	secret := &main.Secret{
		TypeMeta:   generateMetaInformation("Secret", "v1"),
		ObjectMeta: generateObjectMetaInformation(cr.Name+"-secret", cr.Namespace, cr.Labels, cr.Annotations),
		Type:       main.SecretTypeOpaque,
		Data: map[string][]byte{
			"scrape-config.yml": []byte(`
additionalScrapeConfigs:
- job_name: ` + cr.Name + `_server
  static_configs:
  - targets:
    - ` + cr.Name + `-service:` + cr.Spec.ApplicationRef.DeploymentPort + `
`),
		},
	}

	return secret

}
