package utils

import (
	"context"
	"fmt"

	autoscaler "buildpiper.opstreelabs.in/autoscaler/api/v1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetSAccount(cr *autoscaler.CustomAutoScaling) (*corev1.ServiceAccount, error) {
	saName := cr.Name + "-sa"
	logger := k8sLogger(cr.Namespace, saName)
	getOpts := metav1.GetOptions{
		TypeMeta: generateMetaInformation("ServiceAccount", "v1"),
	}
	saInfo, err := generateK8sClient().CoreV1().ServiceAccounts(cr.Namespace).Get(context.TODO(), saName, getOpts)

	if err != nil {

		logger.Error(fmt.Errorf("get serviceAccount failed %s  : %s", saName, err.Error()), "")
		return nil, err
	}

	logger.Info("Prometheus Service Account fetch succesfully ")

	return saInfo, nil
}

func CreateServiceAccount(cr *autoscaler.CustomAutoScaling) (*corev1.ServiceAccount, error) {
	saName := cr.Name + "-sa"
	logger := k8sLogger(cr.Namespace, saName)

	sa := &corev1.ServiceAccount{
		TypeMeta: generateMetaInformation("ServiceAccount", "v1"),
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: cr.Namespace,
		},
	}

	sa, err := generateK8sClient().CoreV1().ServiceAccounts(cr.Namespace).Create(context.Background(), sa, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {

			logger.Error(fmt.Errorf("ServiceAccount %s already exists in namespace  : %s", saName, err.Error()), "")
			return nil, err
		}
		return nil, fmt.Errorf("failed to create ServiceAccount %s in namespace %s: %v", saName, cr.Namespace, err)
	}

	logger.Info("Service Account creation Success")

	return sa, nil
}

func GetClusterRole(cr *autoscaler.CustomAutoScaling) (*rbacv1.ClusterRole, error) {
	roleName := cr.Name + "-clusterrole"
	logger := k8sLogger(cr.Namespace, roleName)
	logger.Info("Fetching clusterRole ......")
	clusterRole, err := generateK8sClient().RbacV1().ClusterRoles().Get(context.TODO(), roleName, metav1.GetOptions{})
	if err != nil {

		logger.Error(fmt.Errorf("error while fetching clusterRole %s  in namespace %s : %s", roleName, cr.Namespace, err.Error()), "")
		return nil, err
	}
	logger.Info("ClusterRole fetched succesfully")
	return clusterRole, nil
}

func CreateClusterRole(cr *autoscaler.CustomAutoScaling) (*rbacv1.ClusterRole, error) {
	roleName := cr.Name + "-clusterrole"
	logger := k8sLogger(cr.Namespace, roleName)
	logger.Info("Fetching clusterRole ......")

	clusterRoleDef := generateClusterDef(roleName, cr.Namespace)
	clusterRole, err := generateK8sClient().RbacV1().ClusterRoles().Create(context.TODO(), clusterRoleDef, metav1.CreateOptions{})
	if err != nil {

		logger.Error(fmt.Errorf("error while fetching clusterRole %s  in namespace %s : %s", roleName, cr.Namespace, err.Error()), "")
		return nil, err
	}
	logger.Info("ClusterRole created succesfully")
	return clusterRole, nil
}

func generateClusterDef(name, namespace string) *rbacv1.ClusterRole {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"nodes", "nodes/metrics", "services", "endpoints", "pods"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps"},
			Verbs:     []string{"get"},
		},
		{
			APIGroups: []string{"networking.k8s.io"},
			Resources: []string{"ingresses"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			NonResourceURLs: []string{"/metrics"},
			Verbs:           []string{"get"},
		},
	}

	clusterRole := &rbacv1.ClusterRole{

		TypeMeta: generateMetaInformation("ClusterRole", "rbac.authorization.k8s.io/v1"),
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Rules: rules,
	}

	return clusterRole

}

func GetRoleBinding(cr *autoscaler.CustomAutoScaling) (*rbacv1.ClusterRoleBinding, error) {
	binding := cr.Name + "-rolebinding"
	logger := k8sLogger(cr.Namespace, binding)

	roleBinding, err := generateK8sClient().RbacV1().ClusterRoleBindings().Get(context.TODO(), binding, metav1.GetOptions{})

	if err != nil {
		logger.Error(fmt.Errorf("unable to fetch rolebinding %s", err.Error()), "")
		return nil, err
	}

	logger.Info("ClusterRoleBinding created succesfully")

	return roleBinding, nil

}

func CreateClusterRoleBinding(cr *autoscaler.CustomAutoScaling) (*rbacv1.ClusterRoleBinding, error) {
	binding := cr.Name + "-rolebinding"
	logger := k8sLogger(cr.Namespace, binding)
	client := generateK8sClient()
	clusterRoleBindingDef := generateClusterRoleBindindingDef(binding, cr.Namespace, cr.Name+"-sa")
	roleBinding, err := client.RbacV1().ClusterRoleBindings().Create(context.TODO(), clusterRoleBindingDef, metav1.CreateOptions{})
	if err != nil {
		logger.Error(fmt.Errorf("unable to create clusterrolebinding %s", err.Error()), "")
		return nil, err
	}
	logger.Info("clusterrolebinding created succesfully")

	return roleBinding, nil
}

func generateClusterRoleBindindingDef(name, namespace, sa string) *rbacv1.ClusterRoleBinding {
	clusterRolebinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: generateMetaInformation("ClusterRoleBinding", " rbac.authorization.k8s.io/v1"),
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa,
				Namespace: namespace,
			},
		},
	}
	return clusterRolebinding
}

func k8sLogger(namespace string, name string) logr.Logger {
	reqLogger := log.WithValues("Request.Service.Namespace", "Request.Service.Name", namespace, name)
	return reqLogger
}
