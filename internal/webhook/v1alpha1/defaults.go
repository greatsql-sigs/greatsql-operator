/*
Copyright 2024 greatsql.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"github.com/greatsql-sigs/greatsql-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	mysqlPort                 = 3306
	defaultTerminationGrace   = int64(60)
	defaultSchedulerName      = "default-scheduler"
	defaultTopologyKey        = "kubernetes.io/hostname"
	startupInitialDelay       = 60
	startupPeriod             = 10
	startupTimeout            = 3
	startupFailureThreshold   = 30
	readinessInitialDelay     = 60
	readinessPeriod           = 5
	readinessTimeout          = 2
	readinessFailureThreshold = 3
	livenessInitialDelay      = 60
	livenessPeriod            = 30
	livenessTimeout           = 5
	livenessFailureThreshold  = 5
)

// defaultMySQLProbe 返回针对 MySQL 3306 端口的默认 TCP 探针（用于 startup/readiness/liveness 的通用参数可调版本）。
func defaultMySQLProbe(initialDelay, period, timeout, failureThreshold int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt32(mysqlPort)},
		},
		InitialDelaySeconds: initialDelay,
		PeriodSeconds:       period,
		TimeoutSeconds:      timeout,
		FailureThreshold:    failureThreshold,
		SuccessThreshold:    1,
	}
}

// defaultMySQLPreStopLifecycle 返回默认的 MySQL 优雅退出 preStop：先 SHUTDOWN 再退出。
func defaultMySQLPreStopLifecycle() *corev1.Lifecycle {
	return &corev1.Lifecycle{
		PreStop: &corev1.LifecycleHandler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"/bin/sh", "-c",
					`if pgrep -f "mysqld" > /dev/null 2>&1; then
  echo "MySQL is running, sending graceful shutdown..."
  if [ -n "$MYSQL_ROOT_PASSWORD" ]; then
    timeout 30 mysql -uroot -p"$MYSQL_ROOT_PASSWORD" -h127.0.0.1 -e "SHUTDOWN;" 2>/dev/null || true
  else
    timeout 30 mysql -uroot -h127.0.0.1 -e "SHUTDOWN;" 2>/dev/null || true
  fi
  for i in $(seq 1 10); do
    if ! pgrep -f "mysqld" > /dev/null 2>&1; then
      echo "MySQL has stopped gracefully"
      exit 0
    fi
    sleep 1
  done
  echo "Sending TERM signal to MySQL..."
  kill -TERM 1 2>/dev/null || true
  sleep 2
else
  echo "MySQL is not running, nothing to shutdown"
fi`,
				},
			},
		},
	}
}

// defaultScheduling 为 Pod 内联的 Scheduling 设置默认值：反亲和、terminationGracePeriod、schedulerName。
func defaultScheduling(s *v1alpha1.Scheduling) {
	if s == nil {
		return
	}
	if s.Affinity == nil {
		key := defaultTopologyKey
		s.Affinity = &v1alpha1.Affinity{TopologyKey: &key}
	} else if s.Affinity.TopologyKey == nil {
		key := defaultTopologyKey
		s.Affinity.TopologyKey = &key
	}
	if s.TerminationGracePeriodSeconds == nil {
		v := defaultTerminationGrace
		s.TerminationGracePeriodSeconds = &v
	}
	if s.SchedulerName == "" {
		s.SchedulerName = defaultSchedulerName
	}
}

// DefaultPodSpec 为 Pod 设置默认值：Scheduling、DnsPolicy、RestartPolicy、探针、preStop。
// 仅当字段为 nil 或零值时设置，不覆盖用户已填写的值。
func DefaultPodSpec(pod *v1alpha1.Pod) {
	if pod == nil {
		return
	}
	// 内联 *Scheduling：若为 nil 则分配并填默认
	if pod.Scheduling == nil {
		pod.Scheduling = &v1alpha1.Scheduling{}
	}
	defaultScheduling(pod.Scheduling)

	if pod.DnsPolicy == "" {
		pod.DnsPolicy = corev1.DNSClusterFirst
	}
	if pod.RestartPolicy == "" {
		pod.RestartPolicy = corev1.RestartPolicyAlways
	}

	// 容器探针与 lifecycle：仅当未设置时填充
	c := &pod.Container
	if c.StartupProbe == nil {
		c.StartupProbe = defaultMySQLProbe(startupInitialDelay, startupPeriod, startupTimeout, startupFailureThreshold)
	}
	if c.ReadinessProbe == nil {
		c.ReadinessProbe = defaultMySQLProbe(readinessInitialDelay, readinessPeriod, readinessTimeout, readinessFailureThreshold)
	}
	if c.LivenessProbe == nil {
		c.LivenessProbe = defaultMySQLProbe(livenessInitialDelay, livenessPeriod, livenessTimeout, livenessFailureThreshold)
	}
	if c.Lifecycle == nil {
		c.Lifecycle = defaultMySQLPreStopLifecycle()
	}
}

// DefaultUpdateStrategy 设置更新策略默认值：Type 为 RollingUpdate。
func DefaultUpdateStrategy(u *v1alpha1.UpdateStrategy) {
	if u == nil {
		return
	}
	if u.Type == "" {
		u.Type = appsv1.RollingUpdateStatefulSetStrategyType
	}
}

// DefaultService 设置 Service 默认值：Type 为 ClusterIP。
func DefaultService(svc *v1alpha1.Service) {
	if svc == nil {
		return
	}
	if svc.Type == "" {
		svc.Type = corev1.ServiceTypeClusterIP
	}
}
