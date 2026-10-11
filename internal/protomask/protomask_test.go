package protomask_test

import (
	"testing"

	racesv1 "github.com/catzkorn/trail-tools/internal/gen/races/v1"
	"github.com/catzkorn/trail-tools/internal/protomask"
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
