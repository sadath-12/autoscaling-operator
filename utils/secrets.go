package utils

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"

	autoscaler "buildpiper.opstreelabs.in/autoscaler/api/v1"
	main "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getSecret(cr *autoscaler.CustomAutoScaling, name string) (*main.Secret, error) {
	secretName := name
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

	filecontent := fmt.Sprintf(`
scrape_configs:
  - job_name: "prometheus"
    static_configs:
      - targets: [%s:%s]
`, cr.Spec.ApplicationRef.DeploymentService, cr.Spec.ApplicationRef.DeploymentPort)

	encodedFileContent := base64.StdEncoding.EncodeToString([]byte(filecontent))
	quotedFileContent := strconv.Quote(encodedFileContent)

	secret := &main.Secret{
		TypeMeta:   generateMetaInformation("Secret", "v1"),
		ObjectMeta: generateObjectMetaInformation(cr.Name+"-secret", cr.Namespace, cr.ObjectMeta.Labels, cr.ObjectMeta.Annotations),
		Type:       main.SecretTypeOpaque,
		Data: map[string][]byte{
			"scrape-config.yml": []byte(quotedFileContent),
		},
	}

	return secret

	// 			"scrape-config.yml": []byte(`
	// additionalScrapeConfigs:
	// - job_name: ` + cr.Name + `_server
	//   static_configs:
	//   - targets:
	//     - ` + cr.Spec.ApplicationRef.DeploymentService + `:` + cr.Spec.ApplicationRef.DeploymentPort + `
	// `),

}

func createAlertConfigSecret(cr *autoscaler.CustomAutoScaling) (*main.Secret, error) {
	secretName := cr.Name + "-alertsecret"
	logger := k8sLogger(cr.Namespace, secretName)

	secretDef := generateAlertsecretDef(cr)

	secret, err := generateK8sClient().CoreV1().Secrets(cr.Namespace).Create(context.TODO(), secretDef, metav1.CreateOptions{})
	if err != nil {

		logger.Error(fmt.Errorf("create alert secret  failed %s  : %s", secretName, err.Error()), "")
		return nil, err
	}

	logger.Info("Alert Secret created succesfully ")

	return secret, nil

}

func generateAlertsecretDef(cr *autoscaler.CustomAutoScaling) *main.Secret {
	secret := &v1.Secret{
		TypeMeta:   generateMetaInformation("Secret", "v1"),
		ObjectMeta: generateObjectMetaInformation(cr.Name+"-alertsecret", cr.Namespace, cr.Labels, cr.Annotations),
		StringData: map[string]string{
			"alertmanager.yaml": `
global:
  resolve_timeout: 5m
inhibit_rules: 
- source_matchers:
  - 'severity = critical'
  target_matchers:
  - 'severity =~ warning|info'
  equal:
  - 'namespace'
  - 'alertname'
- source_matchers:
  - 'severity = warning'
  target_matchers:
  - 'severity = info'
  equal:
  - 'namespace'
  - 'alertname'
- source_matchers:
  - 'alertname = InfoInhibitor'
  target_matchers:
  - 'severity = info'
  equal:
  - 'namespace'
route:
  group_by: ['namespace']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 12h
  receiver: 'webhook_receiver'
  routes:
  - receiver: 'webhook_receiver'
    matchers:
    - alertname =~ "InfoInhibitor|Watchdog"
receivers:
- name: 'webhook_receiver'
  webhook_configs:
  - url: "https://webhook.site/7966a1b6-2633-4a6a-916c-cd74676c9f24"
    send_resolved: false
templates:
- '/etc/alertmanager/config/*.tmpl'`,
		},
	}
	return secret
}
