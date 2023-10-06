//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by controller-gen. DO NOT EDIT.

package rufio

import (
	"k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"net/http"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Connection) DeepCopyInto(out *Connection) {
	*out = *in
	out.AuthSecretRef = in.AuthSecretRef
	if in.ProviderOptions != nil {
		in, out := &in.ProviderOptions, &out.ProviderOptions
		*out = new(ProviderOptions)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Connection.
func (in *Connection) DeepCopy() *Connection {
	if in == nil {
		return nil
	}
	out := new(Connection)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ExperimentalOpts) DeepCopyInto(out *ExperimentalOpts) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ExperimentalOpts.
func (in *ExperimentalOpts) DeepCopy() *ExperimentalOpts {
	if in == nil {
		return nil
	}
	out := new(ExperimentalOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HMACOpts) DeepCopyInto(out *HMACOpts) {
	*out = *in
	if in.Secrets != nil {
		in, out := &in.Secrets, &out.Secrets
		*out = make(HMACSecrets, len(*in))
		for key, val := range *in {
			var outVal []v1.SecretReference
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make([]v1.SecretReference, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HMACOpts.
func (in *HMACOpts) DeepCopy() *HMACOpts {
	if in == nil {
		return nil
	}
	out := new(HMACOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in HMACSecrets) DeepCopyInto(out *HMACSecrets) {
	{
		in := &in
		*out = make(HMACSecrets, len(*in))
		for key, val := range *in {
			var outVal []v1.SecretReference
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make([]v1.SecretReference, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HMACSecrets.
func (in HMACSecrets) DeepCopy() HMACSecrets {
	if in == nil {
		return nil
	}
	out := new(HMACSecrets)
	in.DeepCopyInto(out)
	return *out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IPMITOOLOptions) DeepCopyInto(out *IPMITOOLOptions) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IPMITOOLOptions.
func (in *IPMITOOLOptions) DeepCopy() *IPMITOOLOptions {
	if in == nil {
		return nil
	}
	out := new(IPMITOOLOptions)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IntelAMTOptions) DeepCopyInto(out *IntelAMTOptions) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IntelAMTOptions.
func (in *IntelAMTOptions) DeepCopy() *IntelAMTOptions {
	if in == nil {
		return nil
	}
	out := new(IntelAMTOptions)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Machine) DeepCopyInto(out *Machine) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Machine.
func (in *Machine) DeepCopy() *Machine {
	if in == nil {
		return nil
	}
	out := new(Machine)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Machine) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineCondition) DeepCopyInto(out *MachineCondition) {
	*out = *in
	in.LastUpdateTime.DeepCopyInto(&out.LastUpdateTime)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineCondition.
func (in *MachineCondition) DeepCopy() *MachineCondition {
	if in == nil {
		return nil
	}
	out := new(MachineCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineList) DeepCopyInto(out *MachineList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Machine, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineList.
func (in *MachineList) DeepCopy() *MachineList {
	if in == nil {
		return nil
	}
	out := new(MachineList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *MachineList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineSpec) DeepCopyInto(out *MachineSpec) {
	*out = *in
	in.Connection.DeepCopyInto(&out.Connection)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineSpec.
func (in *MachineSpec) DeepCopy() *MachineSpec {
	if in == nil {
		return nil
	}
	out := new(MachineSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineStatus) DeepCopyInto(out *MachineStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]MachineCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineStatus.
func (in *MachineStatus) DeepCopy() *MachineStatus {
	if in == nil {
		return nil
	}
	out := new(MachineStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProviderOptions) DeepCopyInto(out *ProviderOptions) {
	*out = *in
	if in.IntelAMT != nil {
		in, out := &in.IntelAMT, &out.IntelAMT
		*out = new(IntelAMTOptions)
		**out = **in
	}
	if in.IPMITOOL != nil {
		in, out := &in.IPMITOOL, &out.IPMITOOL
		*out = new(IPMITOOLOptions)
		**out = **in
	}
	if in.Redfish != nil {
		in, out := &in.Redfish, &out.Redfish
		*out = new(RedfishOptions)
		**out = **in
	}
	if in.RPC != nil {
		in, out := &in.RPC, &out.RPC
		*out = new(RPCOptions)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProviderOptions.
func (in *ProviderOptions) DeepCopy() *ProviderOptions {
	if in == nil {
		return nil
	}
	out := new(ProviderOptions)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RPCOptions) DeepCopyInto(out *RPCOptions) {
	*out = *in
	if in.Request != nil {
		in, out := &in.Request, &out.Request
		*out = new(RequestOpts)
		(*in).DeepCopyInto(*out)
	}
	if in.Signature != nil {
		in, out := &in.Signature, &out.Signature
		*out = new(SignatureOpts)
		(*in).DeepCopyInto(*out)
	}
	if in.HMAC != nil {
		in, out := &in.HMAC, &out.HMAC
		*out = new(HMACOpts)
		(*in).DeepCopyInto(*out)
	}
	if in.Experimental != nil {
		in, out := &in.Experimental, &out.Experimental
		*out = new(ExperimentalOpts)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RPCOptions.
func (in *RPCOptions) DeepCopy() *RPCOptions {
	if in == nil {
		return nil
	}
	out := new(RPCOptions)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RedfishOptions) DeepCopyInto(out *RedfishOptions) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RedfishOptions.
func (in *RedfishOptions) DeepCopy() *RedfishOptions {
	if in == nil {
		return nil
	}
	out := new(RedfishOptions)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RequestOpts) DeepCopyInto(out *RequestOpts) {
	*out = *in
	if in.StaticHeaders != nil {
		in, out := &in.StaticHeaders, &out.StaticHeaders
		*out = make(http.Header, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make([]string, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RequestOpts.
func (in *RequestOpts) DeepCopy() *RequestOpts {
	if in == nil {
		return nil
	}
	out := new(RequestOpts)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SignatureOpts) DeepCopyInto(out *SignatureOpts) {
	*out = *in
	if in.IncludedPayloadHeaders != nil {
		in, out := &in.IncludedPayloadHeaders, &out.IncludedPayloadHeaders
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SignatureOpts.
func (in *SignatureOpts) DeepCopy() *SignatureOpts {
	if in == nil {
		return nil
	}
	out := new(SignatureOpts)
	in.DeepCopyInto(out)
	return out
}
