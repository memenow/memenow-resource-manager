package operator

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"testing"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
)

// --- swap helpers ---

func swapLoadChart(t *testing.T, fn func(string) (*chart.Chart, error)) {
	t.Helper()
	orig := loadChart
	loadChart = fn
	t.Cleanup(func() { loadChart = orig })
}

func swapNewHelmConfig(t *testing.T, fn func(*cli.EnvSettings) (*action.Configuration, error)) {
	t.Helper()
	orig := newHelmConfig
	newHelmConfig = fn
	t.Cleanup(func() { newHelmConfig = orig })
}

func swapRunInstall(t *testing.T, fn func(context.Context, HelmInstallRequest, *chart.Chart, *action.Configuration) error) {
	t.Helper()
	orig := runInstall
	runInstall = fn
	t.Cleanup(func() { runInstall = orig })
}

func swapVerifyRel(t *testing.T, fn func(string, string, *action.Configuration) error) {
	t.Helper()
	orig := verifyRel
	verifyRel = fn
	t.Cleanup(func() { verifyRel = orig })
}

func stubAllSuccess(t *testing.T) {
	t.Helper()
	swapNewHelmConfig(t, func(_ *cli.EnvSettings) (*action.Configuration, error) {
		return &action.Configuration{}, nil
	})
	swapLoadChart(t, func(_ string) (*chart.Chart, error) {
		return &chart.Chart{Metadata: &chart.Metadata{Name: "test"}}, nil
	})
	swapRunInstall(t, func(_ context.Context, _ HelmInstallRequest, _ *chart.Chart, _ *action.Configuration) error {
		return nil
	})
	swapVerifyRel(t, func(_, _ string, _ *action.Configuration) error {
		return nil
	})
}

// --- copyValues tests ---

func TestCopyValues_Nil(t *testing.T) {
	t.Parallel()
	got := copyValues(nil)
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestCopyValues_Empty(t *testing.T) {
	t.Parallel()
	src := map[string]interface{}{}
	got := copyValues(src)
	if got == nil {
		t.Fatal("expected non-nil empty map")
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
}

func TestCopyValues_ShallowCopy(t *testing.T) {
	t.Parallel()
	src := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}
	got := copyValues(src)

	if len(got) != len(src) {
		t.Fatalf("length mismatch: got %d, want %d", len(got), len(src))
	}
	for k, v := range src {
		if got[k] != v {
			t.Errorf("key %q: got %v, want %v", k, got[k], v)
		}
	}

	// Mutating copy must not affect original
	got["key3"] = "new"
	if _, exists := src["key3"]; exists {
		t.Error("mutation of copy affected original")
	}
}

// --- validation tests ---

func TestInstallHelmWithContext_MismatchedArrays(t *testing.T) {
	t.Parallel()
	err := InstallHelmWithContext(context.Background(),
		[]string{"a", "b"}, []string{"ns"}, []string{"r1", "r2"}, nil)
	if err == nil {
		t.Fatal("expected error for mismatched arrays")
	}
	if want := "same length"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q should contain %q", err.Error(), want)
	}
}

func TestInstallHelmWithContext_EmptyCharts(t *testing.T) {
	t.Parallel()
	err := InstallHelmWithContext(context.Background(),
		[]string{}, []string{}, []string{}, nil)
	if err == nil {
		t.Fatal("expected error for empty charts")
	}
	if want := "no charts"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q should contain %q", err.Error(), want)
	}
}

func TestInstallHelmRequests_Empty(t *testing.T) {
	t.Parallel()
	err := InstallHelmRequests(context.Background(), []HelmInstallRequest{})
	if err == nil {
		t.Fatal("expected error for empty requests")
	}
	if want := "no installation requests"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q should contain %q", err.Error(), want)
	}
}

// --- mock-based tests ---

func TestInstallHelmWithContext_Success(t *testing.T) {
	stubAllSuccess(t)

	err := InstallHelmWithContext(context.Background(),
		[]string{"chart1"}, []string{"ns1"}, []string{"rel1"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallHelmWithContext_ValuesCopied(t *testing.T) {
	var captured []map[string]interface{}

	swapNewHelmConfig(t, func(_ *cli.EnvSettings) (*action.Configuration, error) {
		return &action.Configuration{}, nil
	})
	swapLoadChart(t, func(_ string) (*chart.Chart, error) {
		return &chart.Chart{Metadata: &chart.Metadata{Name: "test"}}, nil
	})
	swapRunInstall(t, func(_ context.Context, req HelmInstallRequest, _ *chart.Chart, _ *action.Configuration) error {
		captured = append(captured, req.Values)
		return nil
	})
	swapVerifyRel(t, func(_, _ string, _ *action.Configuration) error {
		return nil
	})

	vals := map[string]interface{}{"key": "value"}
	err := InstallHelmWithContext(context.Background(),
		[]string{"c1", "c2"}, []string{"ns1", "ns2"}, []string{"r1", "r2"}, vals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(captured) != 2 {
		t.Fatalf("expected 2 captured requests, got %d", len(captured))
	}

	// Each request should have independent values
	captured[0]["extra"] = "mutated"
	if _, exists := captured[1]["extra"]; exists {
		t.Error("values maps are not independent copies")
	}
}

func TestInstallHelmRequests_InitConfigError(t *testing.T) {
	swapNewHelmConfig(t, func(_ *cli.EnvSettings) (*action.Configuration, error) {
		return nil, errors.New("config init failed")
	})

	err := InstallHelmRequests(context.Background(), []HelmInstallRequest{
		{ChartPath: "c", Namespace: "ns", Release: "r"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "initializing Helm configuration"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q should contain %q", err.Error(), want)
	}
}

func TestInstallHelmRequests_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	swapNewHelmConfig(t, func(_ *cli.EnvSettings) (*action.Configuration, error) {
		return &action.Configuration{}, nil
	})

	err := InstallHelmRequests(ctx, []HelmInstallRequest{
		{ChartPath: "c", Namespace: "ns", Release: "r"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestInstallHelmRequests_InstallError(t *testing.T) {
	swapNewHelmConfig(t, func(_ *cli.EnvSettings) (*action.Configuration, error) {
		return &action.Configuration{}, nil
	})
	swapLoadChart(t, func(_ string) (*chart.Chart, error) {
		return nil, errors.New("chart load failed")
	})

	err := InstallHelmRequests(context.Background(), []HelmInstallRequest{
		{ChartPath: "bad-chart", Namespace: "ns", Release: "r"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "failed to install chart"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q should contain %q", err.Error(), want)
	}
}

func TestInstallHelmRequests_Success(t *testing.T) {
	stubAllSuccess(t)

	err := InstallHelmRequests(context.Background(), []HelmInstallRequest{
		{ChartPath: "c1", Namespace: "ns1", Release: "r1"},
		{ChartPath: "c2", Namespace: "ns2", Release: "r2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallAndVerifyRelease_LoadError(t *testing.T) {
	swapLoadChart(t, func(_ string) (*chart.Chart, error) {
		return nil, errors.New("chart not found")
	})

	err := installAndVerifyRelease(context.Background(),
		HelmInstallRequest{ChartPath: "bad", Namespace: "ns", Release: "r"},
		&action.Configuration{})
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "error loading chart"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q should contain %q", err.Error(), want)
	}
}

func TestInstallAndVerifyRelease_InstallError(t *testing.T) {
	swapLoadChart(t, func(_ string) (*chart.Chart, error) {
		return &chart.Chart{Metadata: &chart.Metadata{Name: "test"}}, nil
	})
	swapRunInstall(t, func(_ context.Context, _ HelmInstallRequest, _ *chart.Chart, _ *action.Configuration) error {
		return errors.New("install failed")
	})

	err := installAndVerifyRelease(context.Background(),
		HelmInstallRequest{ChartPath: "c", Namespace: "ns", Release: "r"},
		&action.Configuration{})
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "install failed"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q should contain %q", err.Error(), want)
	}
}

func TestInstallAndVerifyRelease_VerifyError(t *testing.T) {
	swapLoadChart(t, func(_ string) (*chart.Chart, error) {
		return &chart.Chart{Metadata: &chart.Metadata{Name: "test"}}, nil
	})
	swapRunInstall(t, func(_ context.Context, _ HelmInstallRequest, _ *chart.Chart, _ *action.Configuration) error {
		return nil
	})
	swapVerifyRel(t, func(_, _ string, _ *action.Configuration) error {
		return errors.New("release not found")
	})

	err := installAndVerifyRelease(context.Background(),
		HelmInstallRequest{ChartPath: "c", Namespace: "ns", Release: "r"},
		&action.Configuration{})
	if err == nil {
		t.Fatal("expected error")
	}
	if want := "release not found"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q should contain %q", err.Error(), want)
	}
}

func TestInstallAndVerifyRelease_Success(t *testing.T) {
	swapLoadChart(t, func(_ string) (*chart.Chart, error) {
		return &chart.Chart{Metadata: &chart.Metadata{Name: "test"}}, nil
	})
	swapRunInstall(t, func(_ context.Context, _ HelmInstallRequest, _ *chart.Chart, _ *action.Configuration) error {
		return nil
	})
	swapVerifyRel(t, func(_, _ string, _ *action.Configuration) error {
		return nil
	})

	err := installAndVerifyRelease(context.Background(),
		HelmInstallRequest{ChartPath: "c", Namespace: "ns", Release: "r"},
		&action.Configuration{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- initConfig tests ---

func TestInitConfig_MemoryDriver(t *testing.T) {
	t.Setenv("HELM_DRIVER", "memory")
	settings := cli.New()
	cfg, err := initConfig(settings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil configuration")
	}
}

// --- runInstallAction direct tests ---

func TestRunInstallAction_Success(t *testing.T) {
	t.Parallel()
	cfg := newTestConfig(t)
	testChart := &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    "test-chart",
			Version: "0.1.0",
		},
	}
	req := HelmInstallRequest{
		ChartPath: "test-chart",
		Namespace: "test-ns",
		Release:   "test-rel",
	}

	err := runInstallAction(context.Background(), req, testChart, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunInstallAction_NilValues(t *testing.T) {
	t.Parallel()
	cfg := newTestConfig(t)
	testChart := &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    "nil-vals-chart",
			Version: "0.1.0",
		},
	}
	req := HelmInstallRequest{
		ChartPath: "nil-vals-chart",
		Namespace: "test-ns",
		Release:   "nil-vals-rel",
		Values:    nil,
	}

	err := runInstallAction(context.Background(), req, testChart, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- verifyRelease direct tests (in-memory Helm storage) ---

func newTestConfig(t *testing.T) *action.Configuration {
	t.Helper()
	store := storage.Init(driver.NewMemory())
	registryClient, err := registry.NewClient()
	if err != nil {
		t.Fatalf("failed to create registry client: %v", err)
	}
	return &action.Configuration{
		Releases:       store,
		KubeClient:     &fake.FailingKubeClient{PrintingKubeClient: fake.PrintingKubeClient{Out: io.Discard}},
		Capabilities:   chartutil.DefaultCapabilities,
		RegistryClient: registryClient,
		Log:            log.Printf,
	}
}

func TestVerifyRelease_Found(t *testing.T) {
	t.Parallel()
	cfg := newTestConfig(t)
	rel := &release.Release{
		Name:      "my-release",
		Namespace: "my-ns",
		Info:      &release.Info{Status: release.StatusDeployed},
		Version:   1,
	}
	if err := cfg.Releases.Create(rel); err != nil {
		t.Fatalf("failed to create test release: %v", err)
	}

	if err := verifyRelease("my-release", "my-ns", cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyRelease_NotFound(t *testing.T) {
	t.Parallel()
	cfg := newTestConfig(t)

	err := verifyRelease("missing", "ns", cfg)
	if err == nil {
		t.Fatal("expected error for missing release")
	}
	if want := "not found"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q should contain %q", err.Error(), want)
	}
}

func TestVerifyRelease_WrongNamespace(t *testing.T) {
	t.Parallel()
	cfg := newTestConfig(t)
	rel := &release.Release{
		Name:      "my-release",
		Namespace: "ns-a",
		Info:      &release.Info{Status: release.StatusDeployed},
		Version:   1,
	}
	if err := cfg.Releases.Create(rel); err != nil {
		t.Fatalf("failed to create test release: %v", err)
	}

	err := verifyRelease("my-release", "ns-b", cfg)
	if err == nil {
		t.Fatal("expected error for wrong namespace")
	}
	if want := "not found"; !strings.Contains(err.Error(), want) {
		t.Errorf("error %q should contain %q", err.Error(), want)
	}
}
