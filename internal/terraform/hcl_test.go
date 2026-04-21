package terraform

import (
	"strings"
	"testing"
)

func TestReadAndSetModuleAttribute(t *testing.T) {
	src := []byte(`module "hostedcluster" {
  source = "./modules/cluster"
  worker_count = 3
}
`)
	target := Target{BlockType: "module", Labels: []string{"hostedcluster"}, Attribute: "worker_count"}

	current, err := ReadInt("main.tf", src, target)
	if err != nil {
		t.Fatalf("ReadInt() error = %v", err)
	}
	if current != 3 {
		t.Fatalf("ReadInt() = %d, want 3", current)
	}

	updated, err := SetInt("main.tf", src, target, 5)
	if err != nil {
		t.Fatalf("SetInt() error = %v", err)
	}
	if !strings.Contains(string(updated), "worker_count = 5") {
		t.Fatalf("updated content does not contain new value:\n%s", updated)
	}
}

func TestReadRejectsNonLiteral(t *testing.T) {
	src := []byte(`module "hostedcluster" {
  worker_count = var.worker_count
}
`)
	target := Target{BlockType: "module", Labels: []string{"hostedcluster"}, Attribute: "worker_count"}

	if _, err := ReadInt("main.tf", src, target); err == nil {
		t.Fatal("ReadInt() error = nil, want error")
	}
}
