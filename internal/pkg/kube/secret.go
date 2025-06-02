package kube

import (
	"fmt"

	"github.com/greatsql-sigs/greatsql-operator/internal/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-03-20 14:21:23
 * @file: secret.go
 * @description: secret operation
 */

func NewSecret(name, namespace, key string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				consts.AppKubernetesName: name,
			},
		},
		Data: map[string][]byte{
			key: {},
		},
		Type: corev1.SecretTypeOpaque,
	}
}

func NewSecretEnv(cr client.Object, scheme *runtime.Scheme, name, namespace string, envs []corev1.EnvVar) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				consts.AppKubernetesName: name,
			},
		},
		Data: map[string][]byte{
			consts.MYSQL_ROOT_PASSWORD_KEY: []byte(consts.MYSQL_ROOT_PASSWORD_VALUE),
		},
		Type: corev1.SecretTypeOpaque,
	}

	// 设置 OwnerReference
	if err := controllerutil.SetControllerReference(cr, secret, scheme); err != nil {
		return nil, fmt.Errorf("failed to set controller reference for secret: %w", err)
	}

	return secret, nil
}

func NewSecretEnvFrom(name, namespace string, envFromRefs []corev1.EnvFromSource) *corev1.Secret {
	var secret *corev1.Secret // Declare the "secret" variable
	for _, envFrom := range envFromRefs {
		if envFrom.SecretRef != nil {
			secret = NewSecret(envFrom.SecretRef.Name, name, namespace)
			break
		}
	}
	return secret
}
