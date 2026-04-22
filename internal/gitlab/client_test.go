package gitlab

import (
	"strings"
	"testing"

	"github.com/containeroo/terrascaler/internal/terraform"
)

func TestMergeRequestBranch(t *testing.T) {
	client := &Client{
		cfg: Config{
			MR: MergeRequestConfig{
				BranchPrefix: "/terrascaler/proposals/",
			},
			Target: terraform.Target{
				Attribute: "worker_count",
			},
		},
	}

	got := client.mergeRequestBranch(7)
	want := "terrascaler/proposals-worker-count-to-7"
	if got != want {
		t.Fatalf("mergeRequestBranch() = %q, want %q", got, want)
	}
}

func TestMergeRequestDescription(t *testing.T) {
	client := &Client{
		cfg: Config{
			Branch: "main",
			File:   "main.tf",
			MR: MergeRequestConfig{
				Description: "Please review the proposed scale-up.",
			},
			Target: terraform.Target{
				BlockType: "module",
				Labels:    []string{"hostedcluster"},
				Attribute: "worker_count",
			},
		},
	}

	got := client.mergeRequestDescription(3, 5)
	for _, want := range []string{
		"Please review the proposed scale-up.",
		"increasing worker_count from 3 to 5",
		"Target branch: main",
		"Terraform file: main.tf",
		"Terraform target: module [hostedcluster] worker_count",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("mergeRequestDescription() missing %q in:\n%s", want, got)
		}
	}
}
