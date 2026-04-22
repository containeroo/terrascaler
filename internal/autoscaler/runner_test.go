package autoscaler

import (
	"context"
	"log/slog"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/containeroo/terrascaler/internal/config"
)

func TestRunnerScalesTargetWhenPodsDoNotFit(t *testing.T) {
	now := time.Now()
	target := &fakeTarget{size: 1}
	kube := fake.NewSimpleClientset(
		&corev1.NodeList{Items: []corev1.Node{
			node("worker-1", map[string]string{"role": "worker"}, Resources{MilliCPU: 1000, Memory: 1024 * 1024 * 1024, Pods: 10}),
		}},
		&corev1.PodList{Items: []corev1.Pod{
			runningPod("used", "worker-1", Resources{MilliCPU: 1000, Memory: 128 * 1024 * 1024, Pods: 1}),
			unschedulablePod("pending", now, Resources{MilliCPU: 1000, Memory: 128 * 1024 * 1024, Pods: 1}),
		}},
	)
	runner, err := NewRunner(testConfig(), kube, target, slog.Default())
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}

	if target.size != 2 {
		t.Fatalf("target.size = %d, want 2", target.size)
	}
	if target.setCalls != 1 {
		t.Fatalf("target.setCalls = %d, want 1", target.setCalls)
	}
}

func TestRunnerDoesNotScaleWhenDryRun(t *testing.T) {
	now := time.Now()
	cfg := testConfig()
	cfg.DryRun = true
	target := &fakeTarget{size: 1}
	kube := fake.NewSimpleClientset(
		&corev1.PodList{Items: []corev1.Pod{
			unschedulablePod("pending", now, Resources{MilliCPU: 1000, Memory: 128 * 1024 * 1024, Pods: 1}),
		}},
	)
	runner, err := NewRunner(cfg, kube, target, slog.Default())
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}

	if target.setCalls != 0 {
		t.Fatalf("target.setCalls = %d, want 0", target.setCalls)
	}
}

func testConfig() config.Config {
	return config.Config{
		CheckInterval:    time.Minute,
		ScaleUpCooldown:  0,
		PendingPodMinAge: time.Minute,
		GitLabToken:      "token",
		GitLabProject:    "group/project",
		GitLabBranch:     "main",
		FilePath:         "main.tf",
		BlockType:        "module",
		Labels:           []string{"hostedcluster"},
		Attribute:        "worker_count",
		MinSize:          0,
		MaxSize:          10,
		NodeSelector:     map[string]string{"role": "worker"},
		TemplateCPU:      "1",
		TemplateMemory:   "1Gi",
		TemplatePods:     10,
	}
}

type fakeTarget struct {
	size     int
	setCalls int
}

func (f *fakeTarget) TargetSize(context.Context) (int, error) {
	return f.size, nil
}

func (f *fakeTarget) SetTargetSize(_ context.Context, desired int, validate func(current int, next int) error) (int, int, error) {
	if validate != nil {
		if err := validate(f.size, desired); err != nil {
			return f.size, desired, err
		}
	}
	current := f.size
	f.size = desired
	f.setCalls++
	return current, desired, nil
}
