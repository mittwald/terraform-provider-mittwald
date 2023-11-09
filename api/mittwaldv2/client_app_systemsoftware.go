package mittwaldv2

import (
	"context"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/google/uuid"
	"sort"
	"strings"
)

type DeMittwaldV1AppSystemSoftwareVersionSet []DeMittwaldV1AppSystemSoftwareVersion

func (c *AppClient) GetSystemSoftwareByName(ctx context.Context, name string) (*DeMittwaldV1AppSystemSoftware, bool, error) {
	systemSoftwares, err := c.client.AppListSystemsoftwaresWithResponse(ctx, &AppListSystemsoftwaresParams{})
	if err != nil {
		return nil, false, err
	}

	if systemSoftwares.JSON200 == nil {
		return nil, false, errUnexpectedStatus(systemSoftwares.StatusCode(), systemSoftwares.Body)
	}

	for _, systemSoftware := range *systemSoftwares.JSON200 {
		if strings.EqualFold(systemSoftware.Name, name) {
			return &systemSoftware, true, nil
		}
	}

	return nil, false, nil
}

func (c *AppClient) SelectSystemSoftwareVersion(ctx context.Context, systemSoftwareID, versionSelector string) (DeMittwaldV1AppSystemSoftwareVersionSet, error) {
	versions, err := c.client.AppListSystemsoftwareversionsWithResponse(ctx, uuid.MustParse(systemSoftwareID), &AppListSystemsoftwareversionsParams{VersionRange: &versionSelector})
	if err != nil {
		return nil, err
	}

	if versions.JSON200 == nil {
		return nil, errUnexpectedStatus(versions.StatusCode(), versions.Body)
	}

	set := DeMittwaldV1AppSystemSoftwareVersionSet(*versions.JSON200)
	sort.Sort(set)

	return set, nil
}

func (c *AppClient) GetSystemSoftwareAndVersion(ctx context.Context, systemSoftwareID, systemSoftwareVersionID string) (*DeMittwaldV1AppSystemSoftware, *DeMittwaldV1AppSystemSoftwareVersion, error) {
	systemSoftware, err := c.client.AppGetSystemsoftwareWithResponse(ctx, uuid.MustParse(systemSoftwareID))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get system software '%s': %w", systemSoftwareID, err)
	}

	if systemSoftware.JSON200 == nil {
		return nil, nil, errUnexpectedStatus(systemSoftware.StatusCode(), systemSoftware.Body)
	}

	version, err := c.client.AppGetSystemsoftwareversionWithResponse(ctx, uuid.MustParse(systemSoftwareID), uuid.MustParse(systemSoftwareVersionID))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get system software version '%s': %w", systemSoftwareVersionID, err)
	}

	if version.JSON200 == nil {
		return nil, nil, errUnexpectedStatus(version.StatusCode(), version.Body)
	}

	return systemSoftware.JSON200, version.JSON200, nil
}

func (s DeMittwaldV1AppSystemSoftwareVersionSet) Len() int {
	return len(s)
}

func (s DeMittwaldV1AppSystemSoftwareVersionSet) Less(i, j int) bool {
	verI, _ := semver.NewVersion(s[i].InternalVersion)
	verJ, _ := semver.NewVersion(s[j].InternalVersion)

	return verI.LessThan(verJ)
}

func (s DeMittwaldV1AppSystemSoftwareVersionSet) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s DeMittwaldV1AppSystemSoftwareVersionSet) Recommended() (*DeMittwaldV1AppSystemSoftwareVersion, bool) {
	for _, version := range s {
		if version.Recommended != nil && *version.Recommended {
			return &version, true
		}
	}

	return nil, false
}
