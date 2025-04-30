package kube

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var log = ctrl.Log.WithName("workLoad")

type WorkloadType string

const (
	StatefulSetType WorkloadType = "StatefulSet"
	JobType         WorkloadType = "Job"
	CronJobType     WorkloadType = "CronJob"
)

var workloadGVRMap = map[WorkloadType]schema.GroupVersionResource{
	StatefulSetType: {Group: "apps", Version: "v1", Resource: "statefulsets"},
	JobType:         {Group: "batch", Version: "v1", Resource: "jobs"},
	CronJobType:     {Group: "batch", Version: "v1", Resource: "cronjobs"},
}

// CreateOrPatchWorkload 创建或 Patch StatefulSet、Job、CronJob
func CreateOrPatchWorkload(
	ctx context.Context, c client.Client,
	dynamicClient client.Client, obj client.Object,
	workloadType WorkloadType, namespace string,
	owner metav1.Object, scheme *runtime.Scheme,
	waitReady bool, timeout time.Duration,
) error {
	if err := controllerutil.SetControllerReference(owner, obj, scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	existing := obj.DeepCopyObject().(client.Object)
	err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: obj.GetName()}, existing)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating new workload", "name", obj.GetName(), "type", workloadType)
			if err := c.Create(ctx, obj); err != nil {
				return fmt.Errorf("failed to create %s: %w", workloadType, err)
			}
		} else {
			return fmt.Errorf("failed to get existing %s: %w", workloadType, err)
		}
	} else {
		// 存在，Patch 更新
		log.Info("Patching existing workload", "name", obj.GetName(), "type", workloadType)
		if err := c.Patch(ctx, obj, client.MergeFrom(existing)); err != nil {
			return fmt.Errorf("failed to patch %s: %w", workloadType, err)
		}
	}

	if waitReady {
		return waitForWorkloadReady(ctx, c, obj, workloadType, namespace, timeout)
	}

	return nil
}

// waitForWorkloadReady 等待资源 Ready，并打印状态日志
func waitForWorkloadReady(
	ctx context.Context, c client.Client,
	obj client.Object, workloadType WorkloadType,
	namespace string, timeout time.Duration,
) error {
	name := obj.GetName()

	return wait.PollUntilContextTimeout(ctx, 2*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		switch workloadType {

		case StatefulSetType:
			var sts appsv1.StatefulSet
			if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &sts); err != nil {
				log.Error(err, "Failed to get StatefulSet", "name", name)
				return false, err
			}
			log.Info("Waiting StatefulSet ready", "name", name, "readyReplicas", sts.Status.ReadyReplicas, "desiredReplicas", *sts.Spec.Replicas)
			return sts.Status.ReadyReplicas >= *sts.Spec.Replicas, nil

		case JobType:
			var job batchv1.Job
			if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &job); err != nil {
				log.Error(err, "Failed to get Job", "name", name)
				return false, err
			}
			log.Info("Waiting Job success", "name", name, "succeeded", job.Status.Succeeded)
			return job.Status.Succeeded > 0, nil

		case CronJobType:
			log.Info("CronJob created", "name", name)
			return true, nil

		default:
			return false, fmt.Errorf("unsupported workloadType: %s", workloadType)
		}
	})
}
