package apiext

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/appclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/appv2"
	"github.com/mittwald/api-client-go/pkg/util/pointer"
)

type SystemSoftwareVersionSet []appv2.SystemSoftwareVersion

func (s SystemSoftwareVersionSet) Len() int {
	return len(s)
}

func (s SystemSoftwareVersionSet) Less(i, j int) bool {
	verI, _ := semver.NewVersion(s[i].InternalVersion)
	verJ, _ := semver.NewVersion(s[j].InternalVersion)

	return verI.LessThan(verJ)
}

func (s SystemSoftwareVersionSet) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s SystemSoftwareVersionSet) Recommended() (*appv2.SystemSoftwareVersion, bool) {
	for _, version := range s {
		if version.Recommended != nil && *version.Recommended {
			return &version, true
		}
	}

	return nil, false
}

func (c *appClient) GetSystemsoftwareByName(ctx context.Context, name string) (*appv2.SystemSoftware, bool, error) {
	systemSoftwaresReq := appclientv2.ListSystemsoftwaresRequest{Limit: pointer.To[int64](999)}
	systemSoftwares, _, err := c.ListSystemsoftwares(ctx, systemSoftwaresReq)
	if err != nil {
		return nil, false, err
	}

	for _, systemSoftware := range *systemSoftwares {
		if strings.EqualFold(systemSoftware.Name, name) {
			return &systemSoftware, true, nil
		}
	}

	return nil, false, nil
}

func (c *appClient) GetSystemsoftwareAndVersion(ctx context.Context, systemSoftwareID, systemSoftwareVersionID string) (*appv2.SystemSoftware, *appv2.SystemSoftwareVersion, error) {
	systemSoftwareRequest := appclientv2.GetSystemsoftwareRequest{SystemSoftwareID: systemSoftwareID}
	systemSoftware, _, err := c.GetSystemsoftware(ctx, systemSoftwareRequest)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get system software '%s': %w", systemSoftwareID, err)
	}

	versionRequest := appclientv2.GetSystemsoftwareversionRequest{SystemSoftwareID: systemSoftwareID, SystemSoftwareVersionID: systemSoftwareVersionID}
	version, _, err := c.GetSystemsoftwareversion(ctx, versionRequest)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get system software version '%s': %w", systemSoftwareVersionID, err)
	}

	return systemSoftware, version, nil
}

func (c *appClient) SelectSystemsoftwareVersion(ctx context.Context, systemSoftwareID, versionSelector string) (SystemSoftwareVersionSet, error) {
	versionsRequest := appclientv2.ListSystemsoftwareversionsRequest{SystemSoftwareID: systemSoftwareID, VersionRange: &versionSelector}
	versions, _, err := c.ListSystemsoftwareversions(ctx, versionsRequest)
	if err != nil {
		return nil, err
	}

	set := SystemSoftwareVersionSet(*versions)
	sort.Sort(set)

	return set, nil
}
