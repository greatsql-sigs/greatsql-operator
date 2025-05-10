package kube

import (
	"github.com/greatsql-sigs/greatsql-operator/api/v1alpha1"
	"github.com/greatsql-sigs/greatsql-operator/internal/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func BuildServices(name, nameSpace string, service v1alpha1.ServiceExpose) *corev1.Service {
	// 设置默认端口
	ports := service.Ports
	if len(ports) == 0 {
		ports = []corev1.ServicePort{
			{
				Name:       "mysql",
				Port:       3306,
				TargetPort: intstr.FromInt(3306),
				Protocol:   corev1.ProtocolTCP,
			},
		}
	}

	// 设置默认选择器
	selector := service.Selector
	if selector == nil {
		selector = map[string]string{
			consts.AppKubernetesName: name,
		}
	}

	// 设置默认服务类型
	serviceType := service.Type
	if serviceType == "" {
		serviceType = corev1.ServiceTypeClusterIP
	}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: nameSpace,
			Labels: map[string]string{
				consts.AppKubernetesName: name,
			},
			Annotations: service.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Type:              serviceType,
			Ports:             ports,
			Selector:          selector,
			LoadBalancerClass: service.LoadBalancerClass,
		},
	}
}
