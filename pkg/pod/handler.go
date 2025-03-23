package pod

import (
	"context"
	"fmt"
	"kube-proxy/pkg/resources"
	"log"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Handler struct {
	client    *kubernetes.Clientset
	namespace string
	target    *corev1.Pod
}

func New(cfg *rest.Config, namespace string) (*Handler, error) {
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Handler{client, namespace, nil}, nil
}

func (ph *Handler) Start(ctx context.Context) error {
	id := uuid.New().String()
	var err error

	pod := resources.Pod
	pod.Name = fmt.Sprintf("jump-%s", id)

	ph.target, err = ph.client.CoreV1().Pods(ph.namespace).Create(ctx, pod, v1.CreateOptions{FieldManager: "port-forwarder"})
	if err != nil {
		return err
	}

	cctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	w, err := ph.client.CoreV1().Pods(ph.namespace).Watch(cctx, v1.SingleObject(ph.target.ObjectMeta))
	if err != nil {
		return err
	}

	for e := range w.ResultChan() {
		if p, ok := e.Object.(*corev1.Pod); ok {
			if p.Status.Phase == corev1.PodRunning {
				w.Stop()
			}
		}
	}

	fmt.Printf("pod started - %s/%s\n", ph.Namespace(), ph.Name())

	return nil
}

func (ph *Handler) Delete(ctx context.Context) error {
	log.Println("deleting jump pod")
	return ph.client.CoreV1().Pods(ph.namespace).Delete(ctx, ph.target.Name, v1.DeleteOptions{})
}

func (ph *Handler) Name() string {
	return ph.target.Name
}

func (ph *Handler) Namespace() string {
	return ph.target.Namespace
}
