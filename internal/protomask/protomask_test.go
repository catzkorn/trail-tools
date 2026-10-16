package protomask_test

import (
	"slices"
	"testing"

	racesv1 "github.com/johanbrandhorst/trail-tools/internal/gen/races/v1"
	"github.com/johanbrandhorst/trail-tools/internal/protomask"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestApply(t *testing.T) {
	t.Parallel()
	newDst := func() *racesv1.Race {
		return racesv1.Race_builder{Id: "1", Name: "Old", Location: "Here", Notes: "Notes"}.Build()
	}
	src := racesv1.Race_builder{Id: "2", Name: "New", Location: ""}.Build()

	dst := newDst()
	if err := protomask.Apply(dst, src, &fieldmaskpb.FieldMask{Paths: []string{"name", "location"}}, "id"); err != nil {
		t.Fatal(err)
	}
	if dst.GetId() != "1" || dst.GetName() != "New" || dst.GetLocation() != "" || dst.GetNotes() != "Notes" {
		t.Errorf("unexpected result %v", dst)
	}

	dst = newDst()
	if err := protomask.Apply(dst, src, &fieldmaskpb.FieldMask{Paths: []string{"websiteUrl"}}); err != nil {
		t.Errorf("JSON name path: %v", err)
	}

	dst = newDst()
	if err := protomask.Apply(dst, src, nil, "id"); err != nil {
		t.Fatal(err)
	}
	if dst.GetId() != "1" || dst.GetName() != "New" || dst.GetNotes() != "" {
		t.Errorf("unexpected result for empty mask %v", dst)
	}

	for _, path := range []string{"nope", "id"} {
		if err := protomask.Apply(newDst(), src, &fieldmaskpb.FieldMask{Paths: []string{path}}, "id"); err == nil {
			t.Errorf("expected error for path %q", path)
		}
	}
}

func TestFields(t *testing.T) {
	t.Parallel()
	md := (&racesv1.Race{}).ProtoReflect().Descriptor()
	names := func(mask *fieldmaskpb.FieldMask) ([]string, error) {
		fields, err := protomask.Fields(md, mask, "id")
		var result []string
		for _, fd := range fields {
			result = append(result, string(fd.Name()))
		}
		return result, err
	}

	got, err := names(&fieldmaskpb.FieldMask{Paths: []string{"notes", "websiteUrl", "notes"}})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"notes", "website_url"}; !slices.Equal(got, want) {
		t.Errorf("got fields %v, want %v", got, want)
	}

	got, err = names(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != md.Fields().Len()-1 || slices.Contains(got, "id") {
		t.Errorf("unexpected fields for empty mask %v", got)
	}

	for _, path := range []string{"nope", "id"} {
		if _, err := names(&fieldmaskpb.FieldMask{Paths: []string{path}}); err == nil {
			t.Errorf("expected error for path %q", path)
		}
	}
}
