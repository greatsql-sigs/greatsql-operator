package network

import (
	"fmt"

	"github.com/greatsql-sigs/greatsql-operator/api/v1alpha1"
	"github.com/greatsql-sigs/greatsql-operator/internal/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func BuildServices(cr any, service v1alpha1.Service) (*corev1.Service, error) {
	obj, ok := cr.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("invalid type conversion: expected metav1.Object, got %T", cr)
	}

	name := obj.GetName()
	ns := obj.GetNamespace()

	// 设置默认端口
	ports := []corev1.ServicePort{
		{
			Name:     consts.MySQL,
			Port:     consts.MySQLPort,
			Protocol: corev1.ProtocolTCP,
		},
		{
			Name:     consts.MySQLXProtocol,
			Port:     consts.MySQLXProtocolPort,
			Protocol: corev1.ProtocolTCP,
		},
		{
			Name:     consts.GroupReplication,
			Port:     consts.GroupReplicationPort,
			Protocol: corev1.ProtocolTCP,
		},
	}

	// 内部、外部流量策略默认为本地级别，否则会因为NAT/SNAT引起 客户端连接时断连、连接 reset 、连接超时、多节点访问时会话不稳定等问题
	// 参考：https://kubernetes.io/zh-cn/docs/concepts/services-networking/service/#external-traffic-policy
	internalTrafficPolicy := corev1.ServiceInternalTrafficPolicyLocal
	externalTrafficPolicy := corev1.ServiceExternalTrafficPolicyLocal

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
			Namespace: ns,
			Labels: map[string]string{
				consts.AppKubernetesName: name,
			},
			Annotations: service.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Type:                  serviceType,
			Ports:                 ports,
			InternalTrafficPolicy: &internalTrafficPolicy,
			ExternalTrafficPolicy: externalTrafficPolicy,
			Selector:              selector,
			LoadBalancerClass:     service.LoadBalancerClass,
		},
	}, nil
}
