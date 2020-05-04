// This file is written and used by iter8 project
package v1alpha3

import (
	proto "github.com/gogo/protobuf/proto"
)

// DeepCopy supports using HTTPMatchRequest within kubernetes types, where deepcopy-gen is used.
func (in *HTTPMatchRequest) DeepCopy() *HTTPMatchRequest {
	if in == nil {
		return nil
	}
	p := proto.Clone(in).(*HTTPMatchRequest)
	out := *p
	return &out
}
