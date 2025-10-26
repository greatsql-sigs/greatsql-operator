package kube

import (
	"context"
	"fmt"

	"github.com/greatsql-sigs/greatsql-operator/internal/consts"
	"github.com/greatsql-sigs/greatsql-operator/internal/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NewSecretEnv 创建 Secret，includeReplPassword 控制是否包含复制用户密码
func NewSecretEnv(cr client.Object, scheme *runtime.Scheme, name, namespace string, includeReplPassword bool) (*corev1.Secret, error) {
	// base password data: root password
	secretData := map[string][]byte{
		consts.MYSQL_ROOT_PASSWORD_KEY: []byte(util.GeneratePassword(12)),
	}

	// 只有集群模式才需要复制用户密码
	if includeReplPassword {
		secretData[consts.REPLCATION_CHANNEL_PASSWORD_KEY] = []byte(util.GeneratePassword(16))
	}

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
		Data: secretData,
		Type: corev1.SecretTypeOpaque,
	}

	// 设置 OwnerReference
	if err := controllerutil.SetControllerReference(cr, secret, scheme); err != nil {
		return nil, fmt.Errorf("failed to set controller reference for secret: %w", err)
	}

	return secret, nil
}

// GetPasswordFromSecret 从 Secret 中获取密码
func GetPasswordFromSecret(ctx context.Context, c client.Client, secretName, namespace, passwordKey string) (string, error) {
	secret := &corev1.Secret{}
	if err := c.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret); err != nil {
		return "", fmt.Errorf("failed to get secret %s: %w", secretName, err)
	}

	password, ok := secret.Data[passwordKey]
	if !ok {
		return "", fmt.Errorf("password key %s not found in secret %s", passwordKey, secretName)
	}

	return string(password), nil
}

// DetectUserSecret 检测用户是否通过环境变量提供了 Secret
// 返回：secretName（用户的或默认的）, needCreate（是否需要创建）
func DetectUserSecret(envs []corev1.EnvVar, defaultSecretName, requiredKey string) (secretName string, needCreate bool) {
	// 检查用户是否已经通过环境变量引用了 Secret
	for _, env := range envs {
		if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
			// 必须引用指定的 key（如 MYSQL_ROOT_PASSWORD）
			if env.ValueFrom.SecretKeyRef.Key == requiredKey &&
				env.ValueFrom.SecretKeyRef.Name != "" {
				// 用户已经配置了 Secret 引用，使用用户的 Secret
				return env.ValueFrom.SecretKeyRef.Name, false
			}
		}
	}
	// 使用默认 Secret 名称，需要创建
	return defaultSecretName, true
}
