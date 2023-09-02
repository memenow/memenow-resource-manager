package operator

import (
	"errors"
	"fmt"
	"log"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
)

// InstallHelm installs the charts in the given namespaces with the provided release names
func InstallHelm(charts []string, namespaces []string, release []string) error {
	settings := cli.New()
	actionConfig := initConfig(settings)
	if actionConfig == nil {
		return errors.New("error initializing configuration")
	}

	for i, chart := range charts {
		if err := installAndVerifyRelease(i, chart, namespaces, release, actionConfig); err != nil {
			return err
		}
	}

	return nil
}

// initConfig initializes the Helm configuration and returns an action.Configuration.
// Returns nil if there's an error during initialization.
func initConfig(settings *cli.EnvSettings) *action.Configuration {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(),
		os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		log.Printf("%+v", err)
		return nil
	}
	return actionConfig
}

// installAndVerifyRelease installs the chart located at the given chart path and verifies that the release was deployed successfully.
func installAndVerifyRelease(i int, chart string, namespaces []string, release []string, actionConfig *action.Configuration) error {
	installChart, err := loader.Load(chart)
	if err != nil {
		fmt.Println(chart)
		return fmt.Errorf("error loading chart: %v", err)
	}

	if err := runInstallAction(i, chart, namespaces, release, installChart, actionConfig); err != nil {
		return err
	}

	return verifyRelease(chart, actionConfig)
}

// runInstallAction installs the chart using the Helm install action.
func runInstallAction(i int, chart string, namespaces []string, release []string, installChart *chart.Chart, actionConfig *action.Configuration) error {
	install := action.NewInstall(actionConfig)
	install.ReleaseName = release[i]
	install.Namespace = namespaces[i]
	install.CreateNamespace = true

	_, instErr := install.Run(installChart, map[string]interface{}{
		"image.tag": "1.6.0-rc",
	})

	if instErr != nil {
		return fmt.Errorf("error installing chart: %v", instErr)
	}

	return nil
}

// verifyReleaseByChartName verifies if a release was deployed successfully based on the chart name.
func verifyRelease(chart string, actionConfig *action.Configuration) error {
	list := action.NewList(actionConfig)
	releases, listErr := list.Run()

	if listErr != nil {
		return fmt.Errorf("error listing releases: %v", listErr)
	}

	for _, release := range releases {
		if release.Name == chart {
			fmt.Printf("Release %s deployed successfully!\n", release.Name)
			return nil
		}
	}

	return errors.New("release was not deployed successfully")
}
