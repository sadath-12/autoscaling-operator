package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func generateMetaInformation(resourceKind string, apiVersion string) metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       resourceKind,
		APIVersion: apiVersion,
	}
}

// generateObjectMetaInformation generates the object meta information
func generateObjectMetaInformation(name string, namespace string, labels map[string]string, annotations map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        name,
		Namespace:   namespace,
		Labels:      labels,
		Annotations: annotations,
	} 
}


func generateAlertLabels(name,setupType string,labels map[string]string) map[string]string {
	lbls := map[string]string{
		"app":name,
		"alertManager_setup_type":setupType,
	}

	for k,v := range labels {
		lbls[k]=v 
	}
		
	return lbls
}
func generatePromLabels(name,target string,labels map[string]string) map[string]string {
	lbls := map[string]string{
		"app":name,
		"target_job":target,
	}

	for k,v := range labels {
		lbls[k]=v 
	}
		
	return lbls
}

func generateAlertAnots(app metav1.ObjectMeta) map[string]string {
	anots := map[string]string{
		"buildpiper.opstreelabs.in":"true",
		"buildpiper.opstreelabs.AlertInstance" : app.GetName()+"-alert",
		"buildpiper.opstreelabs.WebhookIntegration":"true",
	}

	for k,v := range app.GetAnnotations() {
		anots[k]=v 
	}

	return filterAnnotations(anots)
}

// filterAnnotations Remove autogenerated annotations which pose no use to downstream objects (Services,Pods,etc)
func filterAnnotations(anots map[string]string) map[string]string {
	// Filter out some problematic annotations we don't want in the template.
	delete(anots, "kubectl.kubernetes.io/last-applied-configuration")
	delete(anots, "banzaicloud.com/last-applied")
	return anots
}