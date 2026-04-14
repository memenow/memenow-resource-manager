// Package operator provides Helm chart installation and lifecycle management.
package operator

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
)

// Testable seams: package-level function variables that tests can override.
var (
	loadChart     = loader.Load
	newHelmConfig = initConfig
	runInstall    = runInstallAction
	verifyRel     = verifyRelease
)

// HelmInstallRequest represents a single Helm installation request
type HelmInstallRequest struct {
	ChartPath string
	Namespace string
	Release   string
	Values    map[string]interface{}
}

// InstallHelmWithContext installs Helm charts with context support for cancellation and timeout.
// Each chart gets its own copy of the provided values.
func InstallHelmWithContext(ctx context.Context, charts []string, namespaces []string, releases []string, values map[string]interface{}) error {
	if len(charts) != len(namespaces) || len(charts) != len(releases) {
		return errors.New("charts, namespaces, and releases arrays must have the same length")
	}

	if len(charts) == 0 {
		return errors.New("no charts provided for installation")
	}

	requests := make([]HelmInstallRequest, len(charts))
	for i := range charts {
		requests[i] = HelmInstallRequest{
			ChartPath: charts[i],
			Namespace: namespaces[i],
			Release:   releases[i],
			Values:    copyValues(values),
		}
	}

	return InstallHelmRequests(ctx, requests)
}

// copyValues returns a shallow copy of the top-level values map so each request
// is independent. Nested maps are still shared; use a deep copy if callers
// mutate nested structures across requests.
func copyValues(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// InstallHelmRequests installs multiple Helm charts from structured requests
func InstallHelmRequests(ctx context.Context, requests []HelmInstallRequest) error {
	if len(requests) == 0 {
		return errors.New("no installation requests provided")
	}

	settings := cli.New()
	actionConfig, err := newHelmConfig(settings)
	if err != nil {
		return fmt.Errorf("error initializing Helm configuration: %w", err)
	}

	for i, req := range requests {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("installation canceled: %w", ctx.Err())
		default:
		}

		log.Printf("Installing chart %d/%d: %s to namespace %s with release name %s",
			i+1, len(requests), req.ChartPath, req.Namespace, req.Release)

		if err := installAndVerifyRelease(ctx, req, actionConfig); err != nil {
			return fmt.Errorf("failed to install chart %s: %w", req.ChartPath, err)
		}

		log.Printf("Successfully installed chart: %s", req.Release)
	}

	return nil
}

// initConfig initializes the Helm configuration and returns an action.Configuration
func initConfig(settings *cli.EnvSettings) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(
		settings.RESTClientGetter(),
		settings.Namespace(),
		os.Getenv("HELM_DRIVER"),
		log.Printf,
	); err != nil {
		return nil, fmt.Errorf("failed to initialize action configuration: %w", err)
	}
	return actionConfig, nil
}

// installAndVerifyRelease installs the chart and verifies deployment success
func installAndVerifyRelease(ctx context.Context, req HelmInstallRequest, actionConfig *action.Configuration) error {
	installChart, err := loadChart(req.ChartPath)
	if err != nil {
		return fmt.Errorf("error loading chart from %s: %w", req.ChartPath, err)
	}

	if err := runInstall(ctx, req, installChart, actionConfig); err != nil {
		return err
	}

	return verifyRel(req.Release, req.Namespace, actionConfig)
}

// runInstallAction installs the chart using the Helm install action
func runInstallAction(ctx context.Context, req HelmInstallRequest, installChart *chart.Chart, actionConfig *action.Configuration) error {
	install := action.NewInstall(actionConfig)
	install.ReleaseName = req.Release
	install.Namespace = req.Namespace
	install.CreateNamespace = true

	// Use provided values or empty map if not provided
	values := req.Values
	if values == nil {
		values = make(map[string]interface{})
	}

	_, err := install.RunWithContext(ctx, installChart, values)
	if err != nil {
		return fmt.Errorf("error installing chart %s: %w", req.ChartPath, err)
	}

	return nil
}

// verifyRelease verifies if a release was deployed successfully
func verifyRelease(releaseName, namespace string, actionConfig *action.Configuration) error {
	list := action.NewList(actionConfig)
	list.All = true           // Include all release statuses
	list.AllNamespaces = true // Search across all namespaces to find the release
	list.Filter = releaseName // Filter by release name for efficiency

	releases, err := list.Run()
	if err != nil {
		return fmt.Errorf("error listing releases: %w", err)
	}

	// Find the release by both name and namespace
	for _, release := range releases {
		if release.Name == releaseName && release.Namespace == namespace {
			log.Printf("Release %s in namespace %s deployed successfully with status: %s",
				releaseName, namespace, release.Info.Status)
			return nil
		}
	}

	return fmt.Errorf("release %s not found in namespace %s after installation", releaseName, namespace)
}
