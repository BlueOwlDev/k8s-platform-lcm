package internal

import (
	"os"
	"strconv"

	"github.com/arminc/k8s-platform-lcm/internal/config"
	"github.com/arminc/k8s-platform-lcm/internal/fetchers"
	"github.com/arminc/k8s-platform-lcm/internal/kubernetes"
	"github.com/arminc/k8s-platform-lcm/internal/scanning"
	"github.com/arminc/k8s-platform-lcm/internal/utils"
	"github.com/olekukonko/tablewriter"
)

// Execute runs all the checks for LCM
func Execute() {
	pods := kubernetes.GetContainersFromNamespaces(config.LcmConfig.Namespaces, config.ConfigFlags.LocalKubernetes)
	info := getLatestVersionsForContainers(pods)
	info = getVulnerabilities(info)
	prettyPrint(info)
}

func FakeExecute(containers []kubernetes.Container) {
	info := getLatestVersionsForContainers(containers)
	info = getVulnerabilities(info)
	prettyPrint(info)
}

func getVulnerabilities(info []ContainerInfo) []ContainerInfo {
	infoWithVul := []ContainerInfo{}
	for _, ci := range info {
		vulnerabilities := scanning.GetVulnerabilities(ci.Container)
		ci.Cves = vulnerabilities
		infoWithVul = append(infoWithVul, ci)
	}
	return infoWithVul
}

// ContainerInfo contains pod information about the container, its version info and security
type ContainerInfo struct {
	Container     kubernetes.Container
	LatestVersion string
	VersionStatus string
	Fetched       bool
	Cves          []string
}

func getLatestVersionsForContainers(containers []kubernetes.Container) []ContainerInfo {
	info := []ContainerInfo{}
	for _, container := range containers {
		registry := determinRegistry(container)
		version := fetchers.GetLatestImageVersionFromRegistry(container.Name, registry)
		info = append(info, ContainerInfo{
			Container:     container,
			LatestVersion: version,
			VersionStatus: utils.DetermineLifeCycleStatus(version, container.Version),
			Fetched:       true,
		})
	}
	return info
}

func determinRegistry(container kubernetes.Container) config.ImageRegistry {
	registry, exists := config.LcmConfig.FindRegistryByOverrideByImage(container.Name)
	if exists {
		return registry
	}

	registry, exists = config.LcmConfig.FindRegistryByOverrideByURL(container.URL)
	if exists {
		return registry
	}

	registry, exists = config.LcmConfig.GetDefaultRegistry()
	if exists {
		return registry
	}

	return config.LcmConfig.FindRegistryByURL(container.URL)
}

func prettyPrint(info []ContainerInfo) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Image", "Version", "Latest", "Cves", "Fetched"})
	table.SetColumnAlignment([]int{3, 1, 1, 3, 3})

	for _, container := range info {
		row := []string{
			container.Container.Name,
			container.Container.Version,
			container.LatestVersion,
			string(len(container.Cves)),
			strconv.FormatBool(container.Fetched),
		}
		table.Append(row)
	}
	table.Render()
}
