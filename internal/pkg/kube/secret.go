package kube

import (
	"github.com/greatsql-sigs/greatsql-operator/internal/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func NewSecretEnv(name, namespace string, envs []corev1.EnvVar) *corev1.Secret {
	data := make(map[string][]byte)
	for _, env := range envs {
		data[env.Name] = []byte(env.ValueFrom.String())
	}

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
		Data: data,
		Type: corev1.SecretTypeOpaque,
	}
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
