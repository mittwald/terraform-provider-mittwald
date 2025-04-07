package apiext

import (
	"context"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/appclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/appv2"
	"sort"
	"strings"
)

type AppVersionSet []appv2.AppVersion

func (s AppVersionSet) Len() int {
	return len(s)
}

func (s AppVersionSet) Less(i, j int) bool {
	verI, _ := semver.NewVersion(s[i].InternalVersion)
	verJ, _ := semver.NewVersion(s[j].InternalVersion)

	return verI.LessThan(verJ)
}

func (s AppVersionSet) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s AppVersionSet) Recommended() (*appv2.AppVersion, bool) {
	for _, version := range s {
		if version.Recommended != nil && *version.Recommended {
			return &version, true
		}
	}

	return nil, false
}

func (s AppVersionSet) FilterByConstraintStr(c string) (AppVersionSet, error) {
	selector, err := semver.NewConstraint(c)
	if err != nil {
		return nil, fmt.Errorf("invalid version selector '%s': %w", c, err)
	}

	return s.FilterByConstraint(selector), nil
}

func (s AppVersionSet) FilterByConstraint(c *semver.Constraints) AppVersionSet {
	var filtered AppVersionSet
	for _, version := range s {
		v, err := semver.NewVersion(version.InternalVersion)
		if err != nil {
			continue
		}
		if c.Check(v) {
			filtered = append(filtered, version)
		}
	}
	return filtered
}

func (c *appClient) GetAppByName(ctx context.Context, name string) (*appv2.App, bool, error) {
	appID, ok := AppNames[strings.ToLower(name)]
	if !ok {
		return nil, false, nil
	}

	appReq := appclientv2.GetAppRequest{AppID: appID}
	app, _, err := c.GetApp(ctx, appReq)
	if err != nil {
		return nil, false, err
	}

	return app, true, nil
}

func (c *appClient) SelectAppVersion(ctx context.Context, appID, versionSelector string) (AppVersionSet, error) {
	versionsRequest := appclientv2.ListAppversionsRequest{AppID: appID}
	versions, _, err := c.ListAppversions(ctx, versionsRequest)
	if err != nil {
		return nil, err
	}

	set := AppVersionSet(*versions)
	sort.Sort(set)

	if versionSelector != "*" && versionSelector != "" {
		filtered, err := set.FilterByConstraintStr(versionSelector)
		if err != nil {
			return nil, fmt.Errorf("failed to filter versions by selector '%s': %w", versionSelector, err)
		}

		set = filtered
	}

	return set, nil
}
