//go:build !ignore_autogenerated

/*

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

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	"github.com/DataDog/chaos-controller/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CPUPressureSpec) DeepCopyInto(out *CPUPressureSpec) {
	*out = *in
	if in.Count != nil {
		in, out := &in.Count, &out.Count
		*out = new(intstr.IntOrString)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CPUPressureSpec.
func (in *CPUPressureSpec) DeepCopy() *CPUPressureSpec {
	if in == nil {
		return nil
	}
	out := new(CPUPressureSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Config) DeepCopyInto(out *Config) {
	*out = *in
	if in.CountTooLarge != nil {
		in, out := &in.CountTooLarge, &out.CountTooLarge
		*out = new(CountTooLargeConfig)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Config.
func (in *Config) DeepCopy() *Config {
	if in == nil {
		return nil
	}
	out := new(Config)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContainerFailureSpec) DeepCopyInto(out *ContainerFailureSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerFailureSpec.
func (in *ContainerFailureSpec) DeepCopy() *ContainerFailureSpec {
	if in == nil {
		return nil
	}
	out := new(ContainerFailureSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CountTooLargeConfig) DeepCopyInto(out *CountTooLargeConfig) {
	*out = *in
	if in.NamespaceThreshold != nil {
		in, out := &in.NamespaceThreshold, &out.NamespaceThreshold
		*out = new(int)
		**out = **in
	}
	if in.ClusterThreshold != nil {
		in, out := &in.ClusterThreshold, &out.ClusterThreshold
		*out = new(int)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CountTooLargeConfig.
func (in *CountTooLargeConfig) DeepCopy() *CountTooLargeConfig {
	if in == nil {
		return nil
	}
	out := new(CountTooLargeConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in DNSDisruptionSpec) DeepCopyInto(out *DNSDisruptionSpec) {
	{
		in := &in
		*out = make(DNSDisruptionSpec, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DNSDisruptionSpec.
func (in DNSDisruptionSpec) DeepCopy() DNSDisruptionSpec {
	if in == nil {
		return nil
	}
	out := new(DNSDisruptionSpec)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DNSRecord) DeepCopyInto(out *DNSRecord) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DNSRecord.
func (in *DNSRecord) DeepCopy() *DNSRecord {
	if in == nil {
		return nil
	}
	out := new(DNSRecord)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DiskFailureSpec) DeepCopyInto(out *DiskFailureSpec) {
	*out = *in
	if in.Paths != nil {
		in, out := &in.Paths, &out.Paths
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.OpenatSyscall != nil {
		in, out := &in.OpenatSyscall, &out.OpenatSyscall
		*out = new(OpenatSyscallSpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DiskFailureSpec.
func (in *DiskFailureSpec) DeepCopy() *DiskFailureSpec {
	if in == nil {
		return nil
	}
	out := new(DiskFailureSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DiskPressureSpec) DeepCopyInto(out *DiskPressureSpec) {
	*out = *in
	in.Throttling.DeepCopyInto(&out.Throttling)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DiskPressureSpec.
func (in *DiskPressureSpec) DeepCopy() *DiskPressureSpec {
	if in == nil {
		return nil
	}
	out := new(DiskPressureSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DiskPressureThrottlingSpec) DeepCopyInto(out *DiskPressureThrottlingSpec) {
	*out = *in
	if in.ReadBytesPerSec != nil {
		in, out := &in.ReadBytesPerSec, &out.ReadBytesPerSec
		*out = new(int)
		**out = **in
	}
	if in.WriteBytesPerSec != nil {
		in, out := &in.WriteBytesPerSec, &out.WriteBytesPerSec
		*out = new(int)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DiskPressureThrottlingSpec.
func (in *DiskPressureThrottlingSpec) DeepCopy() *DiskPressureThrottlingSpec {
	if in == nil {
		return nil
	}
	out := new(DiskPressureThrottlingSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Disruption) DeepCopyInto(out *Disruption) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Disruption.
func (in *Disruption) DeepCopy() *Disruption {
	if in == nil {
		return nil
	}
	out := new(Disruption)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Disruption) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DisruptionCron) DeepCopyInto(out *DisruptionCron) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DisruptionCron.
func (in *DisruptionCron) DeepCopy() *DisruptionCron {
	if in == nil {
		return nil
	}
	out := new(DisruptionCron)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DisruptionCron) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DisruptionCronList) DeepCopyInto(out *DisruptionCronList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]DisruptionCron, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DisruptionCronList.
func (in *DisruptionCronList) DeepCopy() *DisruptionCronList {
	if in == nil {
		return nil
	}
	out := new(DisruptionCronList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DisruptionCronList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DisruptionCronSpec) DeepCopyInto(out *DisruptionCronSpec) {
	*out = *in
	out.TargetResource = in.TargetResource
	in.DisruptionTemplate.DeepCopyInto(&out.DisruptionTemplate)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DisruptionCronSpec.
func (in *DisruptionCronSpec) DeepCopy() *DisruptionCronSpec {
	if in == nil {
		return nil
	}
	out := new(DisruptionCronSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DisruptionCronStatus) DeepCopyInto(out *DisruptionCronStatus) {
	*out = *in
	if in.LastScheduleTime != nil {
		in, out := &in.LastScheduleTime, &out.LastScheduleTime
		*out = (*in).DeepCopy()
	}
	if in.TargetResourcePreviouslyMissing != nil {
		in, out := &in.TargetResourcePreviouslyMissing, &out.TargetResourcePreviouslyMissing
		*out = (*in).DeepCopy()
	}
	if in.History != nil {
		in, out := &in.History, &out.History
		*out = make([]DisruptionCronTrigger, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DisruptionCronStatus.
func (in *DisruptionCronStatus) DeepCopy() *DisruptionCronStatus {
	if in == nil {
		return nil
	}
	out := new(DisruptionCronStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DisruptionCronTrigger) DeepCopyInto(out *DisruptionCronTrigger) {
	*out = *in
	in.CreatedAt.DeepCopyInto(&out.CreatedAt)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DisruptionCronTrigger.
func (in *DisruptionCronTrigger) DeepCopy() *DisruptionCronTrigger {
	if in == nil {
		return nil
	}
	out := new(DisruptionCronTrigger)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DisruptionEvent) DeepCopyInto(out *DisruptionEvent) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DisruptionEvent.
func (in *DisruptionEvent) DeepCopy() *DisruptionEvent {
	if in == nil {
		return nil
	}
	out := new(DisruptionEvent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DisruptionFilter) DeepCopyInto(out *DisruptionFilter) {
	*out = *in
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(labels.Set, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DisruptionFilter.
func (in *DisruptionFilter) DeepCopy() *DisruptionFilter {
	if in == nil {
		return nil
	}
	out := new(DisruptionFilter)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DisruptionList) DeepCopyInto(out *DisruptionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Disruption, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DisruptionList.
func (in *DisruptionList) DeepCopy() *DisruptionList {
	if in == nil {
		return nil
	}
	out := new(DisruptionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DisruptionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DisruptionPulse) DeepCopyInto(out *DisruptionPulse) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DisruptionPulse.
func (in *DisruptionPulse) DeepCopy() *DisruptionPulse {
	if in == nil {
		return nil
	}
	out := new(DisruptionPulse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DisruptionRollout) DeepCopyInto(out *DisruptionRollout) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DisruptionRollout.
func (in *DisruptionRollout) DeepCopy() *DisruptionRollout {
	if in == nil {
		return nil
	}
	out := new(DisruptionRollout)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DisruptionRollout) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DisruptionRolloutList) DeepCopyInto(out *DisruptionRolloutList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]DisruptionRollout, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DisruptionRolloutList.
func (in *DisruptionRolloutList) DeepCopy() *DisruptionRolloutList {
	if in == nil {
		return nil
	}
	out := new(DisruptionRolloutList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DisruptionRolloutList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DisruptionRolloutSpec) DeepCopyInto(out *DisruptionRolloutSpec) {
	*out = *in
	out.TargetResource = in.TargetResource
	in.DisruptionTemplate.DeepCopyInto(&out.DisruptionTemplate)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DisruptionRolloutSpec.
func (in *DisruptionRolloutSpec) DeepCopy() *DisruptionRolloutSpec {
	if in == nil {
		return nil
	}
	out := new(DisruptionRolloutSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DisruptionRolloutStatus) DeepCopyInto(out *DisruptionRolloutStatus) {
	*out = *in
	if in.LatestInitContainersHash != nil {
		in, out := &in.LatestInitContainersHash, &out.LatestInitContainersHash
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.LatestContainersHash != nil {
		in, out := &in.LatestContainersHash, &out.LatestContainersHash
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.LastContainerChangeTime != nil {
		in, out := &in.LastContainerChangeTime, &out.LastContainerChangeTime
		*out = (*in).DeepCopy()
	}
	if in.LastScheduleTime != nil {
		in, out := &in.LastScheduleTime, &out.LastScheduleTime
		*out = (*in).DeepCopy()
	}
	if in.TargetResourcePreviouslyMissing != nil {
		in, out := &in.TargetResourcePreviouslyMissing, &out.TargetResourcePreviouslyMissing
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DisruptionRolloutStatus.
func (in *DisruptionRolloutStatus) DeepCopy() *DisruptionRolloutStatus {
	if in == nil {
		return nil
	}
	out := new(DisruptionRolloutStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DisruptionSpec) DeepCopyInto(out *DisruptionSpec) {
	*out = *in
	if in.Count != nil {
		in, out := &in.Count, &out.Count
		*out = new(intstr.IntOrString)
		**out = **in
	}
	if in.Selector != nil {
		in, out := &in.Selector, &out.Selector
		*out = make(labels.Set, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.AdvancedSelector != nil {
		in, out := &in.AdvancedSelector, &out.AdvancedSelector
		*out = make([]v1.LabelSelectorRequirement, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Filter != nil {
		in, out := &in.Filter, &out.Filter
		*out = new(DisruptionFilter)
		(*in).DeepCopyInto(*out)
	}
	if in.Unsafemode != nil {
		in, out := &in.Unsafemode, &out.Unsafemode
		*out = new(UnsafemodeSpec)
		(*in).DeepCopyInto(*out)
	}
	in.Triggers.DeepCopyInto(&out.Triggers)
	if in.Pulse != nil {
		in, out := &in.Pulse, &out.Pulse
		*out = new(DisruptionPulse)
		**out = **in
	}
	if in.Containers != nil {
		in, out := &in.Containers, &out.Containers
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Network != nil {
		in, out := &in.Network, &out.Network
		*out = new(NetworkDisruptionSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.NodeFailure != nil {
		in, out := &in.NodeFailure, &out.NodeFailure
		*out = new(NodeFailureSpec)
		**out = **in
	}
	if in.ContainerFailure != nil {
		in, out := &in.ContainerFailure, &out.ContainerFailure
		*out = new(ContainerFailureSpec)
		**out = **in
	}
	if in.CPUPressure != nil {
		in, out := &in.CPUPressure, &out.CPUPressure
		*out = new(CPUPressureSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.DiskPressure != nil {
		in, out := &in.DiskPressure, &out.DiskPressure
		*out = new(DiskPressureSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.DiskFailure != nil {
		in, out := &in.DiskFailure, &out.DiskFailure
		*out = new(DiskFailureSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.DNS != nil {
		in, out := &in.DNS, &out.DNS
		*out = make(DNSDisruptionSpec, len(*in))
		copy(*out, *in)
	}
	if in.GRPC != nil {
		in, out := &in.GRPC, &out.GRPC
		*out = new(GRPCDisruptionSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.Reporting != nil {
		in, out := &in.Reporting, &out.Reporting
		*out = new(Reporting)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DisruptionSpec.
func (in *DisruptionSpec) DeepCopy() *DisruptionSpec {
	if in == nil {
		return nil
	}
	out := new(DisruptionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DisruptionStatus) DeepCopyInto(out *DisruptionStatus) {
	*out = *in
	if in.TargetInjections != nil {
		in, out := &in.TargetInjections, &out.TargetInjections
		*out = make(TargetInjections, len(*in))
		for key, val := range *in {
			var outVal map[types.DisruptionKindName]TargetInjection
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = make(TargetInjectorMap, len(*in))
				for key, val := range *in {
					(*out)[key] = *val.DeepCopy()
				}
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DisruptionStatus.
func (in *DisruptionStatus) DeepCopy() *DisruptionStatus {
	if in == nil {
		return nil
	}
	out := new(DisruptionStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DisruptionTrigger) DeepCopyInto(out *DisruptionTrigger) {
	*out = *in
	in.NotBefore.DeepCopyInto(&out.NotBefore)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DisruptionTrigger.
func (in *DisruptionTrigger) DeepCopy() *DisruptionTrigger {
	if in == nil {
		return nil
	}
	out := new(DisruptionTrigger)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DisruptionTriggers) DeepCopyInto(out *DisruptionTriggers) {
	*out = *in
	in.Inject.DeepCopyInto(&out.Inject)
	in.CreatePods.DeepCopyInto(&out.CreatePods)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DisruptionTriggers.
func (in *DisruptionTriggers) DeepCopy() *DisruptionTriggers {
	if in == nil {
		return nil
	}
	out := new(DisruptionTriggers)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EndpointAlteration) DeepCopyInto(out *EndpointAlteration) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EndpointAlteration.
func (in *EndpointAlteration) DeepCopy() *EndpointAlteration {
	if in == nil {
		return nil
	}
	out := new(EndpointAlteration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GRPCDisruptionSpec) DeepCopyInto(out *GRPCDisruptionSpec) {
	*out = *in
	if in.Endpoints != nil {
		in, out := &in.Endpoints, &out.Endpoints
		*out = make([]EndpointAlteration, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GRPCDisruptionSpec.
func (in *GRPCDisruptionSpec) DeepCopy() *GRPCDisruptionSpec {
	if in == nil {
		return nil
	}
	out := new(GRPCDisruptionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in HTTPMethods) DeepCopyInto(out *HTTPMethods) {
	{
		in := &in
		*out = make(HTTPMethods, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HTTPMethods.
func (in HTTPMethods) DeepCopy() HTTPMethods {
	if in == nil {
		return nil
	}
	out := new(HTTPMethods)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in HTTPPaths) DeepCopyInto(out *HTTPPaths) {
	{
		in := &in
		*out = make(HTTPPaths, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HTTPPaths.
func (in HTTPPaths) DeepCopy() HTTPPaths {
	if in == nil {
		return nil
	}
	out := new(HTTPPaths)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HostRecordPair) DeepCopyInto(out *HostRecordPair) {
	*out = *in
	out.Record = in.Record
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HostRecordPair.
func (in *HostRecordPair) DeepCopy() *HostRecordPair {
	if in == nil {
		return nil
	}
	out := new(HostRecordPair)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NetworkDisruptionCloudServiceSpec) DeepCopyInto(out *NetworkDisruptionCloudServiceSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NetworkDisruptionCloudServiceSpec.
func (in *NetworkDisruptionCloudServiceSpec) DeepCopy() *NetworkDisruptionCloudServiceSpec {
	if in == nil {
		return nil
	}
	out := new(NetworkDisruptionCloudServiceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NetworkDisruptionCloudSpec) DeepCopyInto(out *NetworkDisruptionCloudSpec) {
	*out = *in
	if in.AWSServiceList != nil {
		in, out := &in.AWSServiceList, &out.AWSServiceList
		*out = new([]NetworkDisruptionCloudServiceSpec)
		if **in != nil {
			in, out := *in, *out
			*out = make([]NetworkDisruptionCloudServiceSpec, len(*in))
			copy(*out, *in)
		}
	}
	if in.GCPServiceList != nil {
		in, out := &in.GCPServiceList, &out.GCPServiceList
		*out = new([]NetworkDisruptionCloudServiceSpec)
		if **in != nil {
			in, out := *in, *out
			*out = make([]NetworkDisruptionCloudServiceSpec, len(*in))
			copy(*out, *in)
		}
	}
	if in.DatadogServiceList != nil {
		in, out := &in.DatadogServiceList, &out.DatadogServiceList
		*out = new([]NetworkDisruptionCloudServiceSpec)
		if **in != nil {
			in, out := *in, *out
			*out = make([]NetworkDisruptionCloudServiceSpec, len(*in))
			copy(*out, *in)
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NetworkDisruptionCloudSpec.
func (in *NetworkDisruptionCloudSpec) DeepCopy() *NetworkDisruptionCloudSpec {
	if in == nil {
		return nil
	}
	out := new(NetworkDisruptionCloudSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NetworkDisruptionHostSpec) DeepCopyInto(out *NetworkDisruptionHostSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NetworkDisruptionHostSpec.
func (in *NetworkDisruptionHostSpec) DeepCopy() *NetworkDisruptionHostSpec {
	if in == nil {
		return nil
	}
	out := new(NetworkDisruptionHostSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NetworkDisruptionServicePortSpec) DeepCopyInto(out *NetworkDisruptionServicePortSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NetworkDisruptionServicePortSpec.
func (in *NetworkDisruptionServicePortSpec) DeepCopy() *NetworkDisruptionServicePortSpec {
	if in == nil {
		return nil
	}
	out := new(NetworkDisruptionServicePortSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NetworkDisruptionServiceSpec) DeepCopyInto(out *NetworkDisruptionServiceSpec) {
	*out = *in
	if in.Ports != nil {
		in, out := &in.Ports, &out.Ports
		*out = make([]NetworkDisruptionServicePortSpec, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NetworkDisruptionServiceSpec.
func (in *NetworkDisruptionServiceSpec) DeepCopy() *NetworkDisruptionServiceSpec {
	if in == nil {
		return nil
	}
	out := new(NetworkDisruptionServiceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NetworkDisruptionSpec) DeepCopyInto(out *NetworkDisruptionSpec) {
	*out = *in
	if in.Hosts != nil {
		in, out := &in.Hosts, &out.Hosts
		*out = make([]NetworkDisruptionHostSpec, len(*in))
		copy(*out, *in)
	}
	if in.AllowedHosts != nil {
		in, out := &in.AllowedHosts, &out.AllowedHosts
		*out = make([]NetworkDisruptionHostSpec, len(*in))
		copy(*out, *in)
	}
	if in.Services != nil {
		in, out := &in.Services, &out.Services
		*out = make([]NetworkDisruptionServiceSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Cloud != nil {
		in, out := &in.Cloud, &out.Cloud
		*out = new(NetworkDisruptionCloudSpec)
		(*in).DeepCopyInto(*out)
	}
	if in.DeprecatedPort != nil {
		in, out := &in.DeprecatedPort, &out.DeprecatedPort
		*out = new(int)
		**out = **in
	}
	if in.HTTP != nil {
		in, out := &in.HTTP, &out.HTTP
		*out = new(NetworkHTTPFilters)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NetworkDisruptionSpec.
func (in *NetworkDisruptionSpec) DeepCopy() *NetworkDisruptionSpec {
	if in == nil {
		return nil
	}
	out := new(NetworkDisruptionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NetworkHTTPFilters) DeepCopyInto(out *NetworkHTTPFilters) {
	*out = *in
	if in.Methods != nil {
		in, out := &in.Methods, &out.Methods
		*out = make(HTTPMethods, len(*in))
		copy(*out, *in)
	}
	if in.Paths != nil {
		in, out := &in.Paths, &out.Paths
		*out = make(HTTPPaths, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NetworkHTTPFilters.
func (in *NetworkHTTPFilters) DeepCopy() *NetworkHTTPFilters {
	if in == nil {
		return nil
	}
	out := new(NetworkHTTPFilters)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeFailureSpec) DeepCopyInto(out *NodeFailureSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeFailureSpec.
func (in *NodeFailureSpec) DeepCopy() *NodeFailureSpec {
	if in == nil {
		return nil
	}
	out := new(NodeFailureSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OpenatSyscallSpec) DeepCopyInto(out *OpenatSyscallSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OpenatSyscallSpec.
func (in *OpenatSyscallSpec) DeepCopy() *OpenatSyscallSpec {
	if in == nil {
		return nil
	}
	out := new(OpenatSyscallSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Reporting) DeepCopyInto(out *Reporting) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Reporting.
func (in *Reporting) DeepCopy() *Reporting {
	if in == nil {
		return nil
	}
	out := new(Reporting)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetInjection) DeepCopyInto(out *TargetInjection) {
	*out = *in
	in.Since.DeepCopyInto(&out.Since)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetInjection.
func (in *TargetInjection) DeepCopy() *TargetInjection {
	if in == nil {
		return nil
	}
	out := new(TargetInjection)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in TargetInjections) DeepCopyInto(out *TargetInjections) {
	{
		in := &in
		*out = make(TargetInjections, len(*in))
		for key, val := range *in {
			var outVal map[types.DisruptionKindName]TargetInjection
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = make(TargetInjectorMap, len(*in))
				for key, val := range *in {
					(*out)[key] = *val.DeepCopy()
				}
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetInjections.
func (in TargetInjections) DeepCopy() TargetInjections {
	if in == nil {
		return nil
	}
	out := new(TargetInjections)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in TargetInjectorMap) DeepCopyInto(out *TargetInjectorMap) {
	{
		in := &in
		*out = make(TargetInjectorMap, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetInjectorMap.
func (in TargetInjectorMap) DeepCopy() TargetInjectorMap {
	if in == nil {
		return nil
	}
	out := new(TargetInjectorMap)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TargetResourceSpec) DeepCopyInto(out *TargetResourceSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TargetResourceSpec.
func (in *TargetResourceSpec) DeepCopy() *TargetResourceSpec {
	if in == nil {
		return nil
	}
	out := new(TargetResourceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UnsafemodeSpec) DeepCopyInto(out *UnsafemodeSpec) {
	*out = *in
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = new(Config)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UnsafemodeSpec.
func (in *UnsafemodeSpec) DeepCopy() *UnsafemodeSpec {
	if in == nil {
		return nil
	}
	out := new(UnsafemodeSpec)
	in.DeepCopyInto(out)
	return out
}
