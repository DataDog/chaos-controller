// Package install installs the experimental API group, making it available as
// an option to all of the API encoding/decoding machinery.
package v1beta1

import (
	chaosv1beta1 "github.com/DataDog/chaos-controller/api/v1beta1"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

// Install registers the API group and adds types to a scheme
func Install(scheme *runtime.Scheme) {
	utilruntime.Must(chaosv1beta1.AddToScheme(scheme))
	utilruntime.Must(scheme.SetVersionPriority(v1beta1.SchemeGroupVersion))
}
