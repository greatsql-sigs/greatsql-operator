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

var _ = Describe("GroupReplicationCluster Webhook", func() {
	var (
		ctx       context.Context
		obj       *databasev1alpha1.GroupReplicationCluster
		oldObj    *databasev1alpha1.GroupReplicationCluster
		validator GroupReplicationClusterCustomValidator
		defaulter GroupReplicationClusterCustomDefaulter
	)

	BeforeEach(func() {
		ctx = context.Background()
		obj = &databasev1alpha1.GroupReplicationCluster{}
		oldObj = &databasev1alpha1.GroupReplicationCluster{}
		validator = GroupReplicationClusterCustomValidator{}
		defaulter = GroupReplicationClusterCustomDefaulter{}
	})

	Context("Defaulting", func() {
		It("should default spec.mode to single when empty", func() {
			obj.Spec.Mode = ""
			Expect(defaulter.Default(ctx, obj)).To(Succeed())
			Expect(obj.Spec.Mode).To(Equal(databasev1alpha1.ClusterModeSingle))
		})

		It("should default spec.member[].size to 1 when nil", func() {
			obj.Spec.Member = []databasev1alpha1.Member{
				{Role: databasev1alpha1.PrimaryRole},
			}
			Expect(defaulter.Default(ctx, obj)).To(Succeed())
			Expect(obj.Spec.Member[0].Size).NotTo(BeNil())
			Expect(*obj.Spec.Member[0].Size).To(Equal(int32(1)))
		})

		It("should correct member size when <= 0", func() {
			zero := int32(0)
			obj.Spec.Member = []databasev1alpha1.Member{
				{Role: databasev1alpha1.PrimaryRole, Size: &zero},
			}
			Expect(defaulter.Default(ctx, obj)).To(Succeed())
			Expect(*obj.Spec.Member[0].Size).To(Equal(int32(1)))
		})
	})

	Context("Validating Create", func() {
		It("should reject empty spec.member", func() {
			obj.Spec.Mode = databasev1alpha1.ClusterModeSingle
			obj.Spec.Member = nil
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one member"))
		})

		It("should reject member with empty role", func() {
			obj.Spec.Mode = databasev1alpha1.ClusterModeSingle
			obj.Spec.Member = []databasev1alpha1.Member{{Role: ""}}
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("role is required"))
		})

		It("should reject single mode with no primary", func() {
			obj.Spec.Mode = databasev1alpha1.ClusterModeSingle
			obj.Spec.Member = []databasev1alpha1.Member{
				{Role: databasev1alpha1.SecondaryRole, Size: int32Ptr(int32(1))},
			}
			_, err := validator.ValidateCreate(ctx, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exactly 1 primary"))
		})

		It("should accept valid single mode spec", func() {
			obj.Spec.Mode = databasev1alpha1.ClusterModeSingle
			obj.Spec.Member = []databasev1alpha1.Member{
				{Role: databasev1alpha1.PrimaryRole, Size: int32Ptr(int32(1))},
				{Role: databasev1alpha1.SecondaryRole, Size: int32Ptr(int32(2))},
			}
			warnings, err := validator.ValidateCreate(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeNil())
		})
	})

	Context("Validating Update", func() {
		It("should reject change of spec.mode", func() {
			oldObj.Spec.Mode = databasev1alpha1.ClusterModeSingle
			oldObj.Spec.Member = []databasev1alpha1.Member{
				{Role: databasev1alpha1.PrimaryRole, Size: int32Ptr(int32(1))},
			}
			obj.Spec.Mode = databasev1alpha1.ClusterModeMultiple
			obj.Spec.Member = []databasev1alpha1.Member{
				{Role: databasev1alpha1.PrimaryRole, Size: int32Ptr(int32(1))},
				{Role: databasev1alpha1.PrimaryRole, Size: int32Ptr(int32(1))},
				{Role: databasev1alpha1.SecondaryRole, Size: int32Ptr(int32(1))},
			}
			_, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("immutable"))
		})

		It("should accept update when mode unchanged", func() {
			oldObj.Spec.Mode = databasev1alpha1.ClusterModeSingle
			oldObj.Spec.Member = []databasev1alpha1.Member{
				{Role: databasev1alpha1.PrimaryRole, Size: int32Ptr(int32(1))},
			}
			obj.Spec.Mode = databasev1alpha1.ClusterModeSingle
			obj.Spec.Member = []databasev1alpha1.Member{
				{Role: databasev1alpha1.PrimaryRole, Size: int32Ptr(int32(1))},
				{Role: databasev1alpha1.SecondaryRole, Size: int32Ptr(int32(2))},
			}
			warnings, err := validator.ValidateUpdate(ctx, oldObj, obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeNil())
		})
	})

	Context("Validating Delete", func() {
		It("should allow delete", func() {
			obj.Spec.Mode = databasev1alpha1.ClusterModeSingle
			obj.Spec.Member = []databasev1alpha1.Member{
				{Role: databasev1alpha1.PrimaryRole, Size: int32Ptr(int32(1))},
			}
			warnings, err := validator.ValidateDelete(ctx, obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeNil())
		})
	})
})
