// +build !ignore_autogenerated

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
	authenticationv1 "k8s.io/api/authentication/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CPUPressureSpec) DeepCopyInto(out *CPUPressureSpec) {
	*out = *in
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
		**out = **in
	}
	if in.DiskPressure != nil {
		in, out := &in.DiskPressure, &out.DiskPressure
		*out = new(DiskPressureSpec)
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
	if in.Targets != nil {
		in, out := &in.Targets, &out.Targets
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.UserInfo != nil {
		in, out := &in.UserInfo, &out.UserInfo
		*out = new(authenticationv1.UserInfo)
		(*in).DeepCopyInto(*out)
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
func (in *NetworkDisruptionServiceSpec) DeepCopyInto(out *NetworkDisruptionServiceSpec) {
	*out = *in
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
		copy(*out, *in)
	}
	if in.DeprecatedPort != nil {
		in, out := &in.DeprecatedPort, &out.DeprecatedPort
		*out = new(int)
		**out = **in
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
