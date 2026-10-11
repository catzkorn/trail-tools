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
	fields := dstRefl.Descriptor().Fields()
	var toCopy []protoreflect.FieldDescriptor
	if len(mask.GetPaths()) == 0 {
		for i := range fields.Len() {
			toCopy = append(toCopy, fields.Get(i))
		}
	} else {
		for _, path := range mask.GetPaths() {
			fd := fields.ByName(protoreflect.Name(path))
			if fd == nil {
				fd = fields.ByJSONName(path)
			}
			if fd == nil || slices.Contains(keep, fd.Name()) {
				return fmt.Errorf("invalid update mask path %q", path)
			}
			toCopy = append(toCopy, fd)
		}
	}
	for _, fd := range toCopy {
		if slices.Contains(keep, fd.Name()) {
			continue
		}
		if srcRefl.Has(fd) {
			dstRefl.Set(fd, srcRefl.Get(fd))
		} else {
			dstRefl.Clear(fd)
		}
	}
	return nil
}
