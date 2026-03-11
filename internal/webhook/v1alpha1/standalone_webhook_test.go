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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	databasev1alpha1 "github.com/greatsql-sigs/greatsql-operator/api/v1alpha1"
)

var _ = Describe("Standalone Webhook", func() {
	var (
		ctx       context.Context
		obj       *databasev1alpha1.Standalone
		oldObj    *databasev1alpha1.Standalone
		validator StandaloneCustomValidator
		defaulter StandaloneCustomDefaulter
	)

	BeforeEach(func() {
		ctx = context.Background()
		obj = &databasev1alpha1.Standalone{}
		oldObj = &databasev1alpha1.Standalone{}
		// StandaloneSpec 内联 *Pod，访问 Container 前需初始化 Pod
		obj.Spec.Pod = &databasev1alpha1.Pod{}
		oldObj.Spec.Pod = &databasev1alpha1.Pod{}
		validator = StandaloneCustomValidator{}
		defaulter = StandaloneCustomDefaulter{}
	})

	Context("Defaulting", func() {
		It("should default spec.size to 1 when nil", func() {
			obj.Spec.Size = nil
			Expect(defaulter.Default(ctx, obj)).To(Succeed())
			Expect(obj.Spec.Size).NotTo(BeNil())
			Expect(*obj.Spec.Size).To(Equal(int32(1)))
		})

		It("should correct spec.size when <= 0", func() {
			zero := int32(0)
			obj.Spec.Size = &zero
			Expect(defaulter.Default(ctx, obj)).To(Succeed())
			Expect(*obj.Spec.Size).To(Equal(int32(1)))
		})

		It("should default spec.Pod to empty when nil", func() {
			obj.Spec.Pod = nil
			Expect(defaulter.Default(ctx, obj)).To(Succeed())
			Expect(obj.Spec.Pod).NotTo(BeNil())
		})
	})

	Context("Validating Create", func() {
		It("should reject when spec.Pod is nil (no pod/container config)", func() {
			obj.Spec.Size = int32Ptr(1)
			obj.Spec.Pod = nil
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("container"))
		})

		It("should reject when spec.size is zero or nil before defaulting", func() {
			obj.Spec.Size = nil
			obj.Spec.Container = databasev1alpha1.Container{Name: "mysql", Image: "greatsql:8.0"}
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("size must be positive"))
		})

		It("should reject when container name is empty", func() {
			obj.Spec.Size = int32Ptr(1)
			obj.Spec.Container = databasev1alpha1.Container{Name: "", Image: "greatsql:8.0"}
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("container.name is required"))
		})

		It("should reject when container image is empty", func() {
			obj.Spec.Size = int32Ptr(1)
			obj.Spec.Container = databasev1alpha1.Container{Name: "mysql", Image: ""}
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("container.image is required"))
		})

		It("should accept valid spec", func() {
			obj.Spec.Size = int32Ptr(1)
			obj.Spec.Container = databasev1alpha1.Container{Name: "mysql", Image: "greatsql:8.0"}
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeNil())
		})
	})

	Context("Validating Update", func() {
		It("should accept valid update", func() {
			oldObj.Spec.Size = int32Ptr(1)
			oldObj.Spec.Container = databasev1alpha1.Container{Name: "mysql", Image: "greatsql:8.0"}
			obj.Spec.Size = int32Ptr(2)
			obj.Spec.Container = databasev1alpha1.Container{Name: "mysql", Image: "greatsql:8.0.32"}
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeNil())
		})
	})

	Context("Validating Delete", func() {
		It("should allow delete", func() {
			obj.Spec.Size = int32Ptr(1)
			obj.Spec.Container = databasev1alpha1.Container{Name: "mysql", Image: "greatsql:8.0"}
			warnings, err := validator.ValidateDelete(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeNil())
		})
	})
})
