package resources

import (
	_ "embed"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/client-go/kubernetes/scheme"
)

//go:embed pod.yaml
var pod []byte

var Pod *corev1.Pod

func init() {
	decode := scheme.Codecs.UniversalDeserializer().Decode

	obj, _, _ := decode(pod, nil, nil)
	Pod = obj.(*corev1.Pod)
}
