// Package protomask applies update masks to protobuf messages.
package protomask

import (
	"fmt"
	"slices"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// Apply copies the top level fields named by the mask from src to dst. Paths
// may be field names (snake_case) or JSON names (lowerCamelCase). If the mask
// is empty, all fields are copied. Fields named in keep are never copied.
func Apply(dst, src proto.Message, mask *fieldmaskpb.FieldMask, keep ...protoreflect.Name) error {
	dstRefl, srcRefl := dst.ProtoReflect(), src.ProtoReflect()
	fields, err := Fields(dstRefl.Descriptor(), mask, keep...)
	if err != nil {
		return err
	}
	for _, fd := range fields {
		if srcRefl.Has(fd) {
			dstRefl.Set(fd, srcRefl.Get(fd))
		} else {
			dstRefl.Clear(fd)
		}
	}
	return nil
}

// Fields returns the top level fields of a message named by the mask, in
// mask order without duplicates. Paths may be field names (snake_case) or
// JSON names (lowerCamelCase). If the mask is empty, it returns all fields
// except those named in keep. Naming a field in keep is an error.
func Fields(md protoreflect.MessageDescriptor, mask *fieldmaskpb.FieldMask, keep ...protoreflect.Name) ([]protoreflect.FieldDescriptor, error) {
	fields := md.Fields()
	var result []protoreflect.FieldDescriptor
	if len(mask.GetPaths()) == 0 {
		for i := range fields.Len() {
			if fd := fields.Get(i); !slices.Contains(keep, fd.Name()) {
				result = append(result, fd)
			}
		}
		return result, nil
	}
	for _, path := range mask.GetPaths() {
		fd := fields.ByName(protoreflect.Name(path))
		if fd == nil {
			fd = fields.ByJSONName(path)
		}
		if fd == nil || slices.Contains(keep, fd.Name()) {
			return nil, fmt.Errorf("invalid update mask path %q", path)
		}
		if !slices.Contains(result, fd) {
			result = append(result, fd)
		}
	}
	return result, nil
}
