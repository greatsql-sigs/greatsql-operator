package kube

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

/**
 * @author: HuaiAn xu
 * @date: 2024-03-18 23:52:44
 * @file: finalizer.go
 * @description: resource finalizer
 */
func HandleFinalizerWithCleanup[T client.Object](
	ctx context.Context, c client.Client,
	obj T, finalizer string,
	log logr.Logger, cleanupFn func(context.Context, T) error) error {

	if reflect.ValueOf(obj).IsNil() {
		log.Error(nil, "object is nil")
		return fmt.Errorf("object is nil")
	}

	log = log.WithValues(
		"kind", obj.GetObjectKind().GroupVersionKind().Kind,
		"name", obj.GetName(),
		"namespace", obj.GetNamespace(),
	)

	if obj.GetDeletionTimestamp() != nil {
		log.Info("start to handle resource deletion")
		if controllerutil.ContainsFinalizer(obj, finalizer) {
			if err := cleanupFn(ctx, obj); err != nil {
				log.Error(err, "cleanup function failed")
				return err
			}

			log.Info("remove finalizer")
			controllerutil.RemoveFinalizer(obj, finalizer)
			if err := c.Update(ctx, obj); err != nil {
				return err
			}
		} else {
			log.Info("object does not contain finalizer, skip")
		}
		return nil
	}

	if !controllerutil.ContainsFinalizer(obj, finalizer) {
		log.Info("add finalizer")
		controllerutil.AddFinalizer(obj, finalizer)
		if err := c.Update(ctx, obj); err != nil {
			return err
		}
	} else {
		log.Info("object already contains finalizer, skip")
	}

	return nil
}
