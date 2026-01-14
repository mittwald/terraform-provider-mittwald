package articledatasource

import (
	"fmt"
	"strings"

	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/articlev2"
)

// formatMultipleMatchesError creates a comprehensive error message when multiple articles match the filter criteria.
func formatMultipleMatchesError(articles []articlev2.ReadableArticle, filterTags, filterTemplate, filterOrderable []string, filterAttributes map[string]string, filterIDPattern string) string {
	errorDetail := fmt.Sprintf("Found %d articles matching the specified filters. Please refine your filters to match exactly one article.\n\n", len(articles))
	errorDetail += formatAppliedFiltersSection(filterTags, filterTemplate, filterOrderable, filterAttributes, filterIDPattern)
	errorDetail += formatMatchingArticlesList(articles)
	errorDetail += formatFilterSuggestionsSection()
	return errorDetail
}

// formatAppliedFiltersSection formats the applied filters section of the error message.
func formatAppliedFiltersSection(filterTags, filterTemplate, filterOrderable []string, filterAttributes map[string]string, filterIDPattern string) string {
	section := "Applied filters:\n"

	if len(filterTags) > 0 {
		section += fmt.Sprintf("  - Tags: %v\n", filterTags)
	}
	if len(filterTemplate) > 0 {
		section += fmt.Sprintf("  - Templates: %v\n", filterTemplate)
	}
	if len(filterOrderable) > 0 {
		section += fmt.Sprintf("  - Orderable statuses: %v\n", filterOrderable)
	}
	if len(filterAttributes) > 0 {
		section += "  - Attributes:\n"
		for key, value := range filterAttributes {
			section += fmt.Sprintf("      %s = %s\n", key, value)
		}
	}
	if filterIDPattern != "" {
		section += fmt.Sprintf("  - ID pattern: %s\n", filterIDPattern)
	}
	if len(filterTags) == 0 && len(filterTemplate) == 0 && len(filterOrderable) == 0 && len(filterAttributes) == 0 && filterIDPattern == "" {
		section += "  - No filters applied (matching all articles)\n"
	}

	return section + "\n"
}

// formatMatchingArticlesList formats the list of matching articles.
func formatMatchingArticlesList(articles []articlev2.ReadableArticle) string {
	section := "Matching articles:\n"

	for i, article := range articles {
		section += formatArticleSummary(i+1, article)

		// Limit to showing first 10 articles to avoid overwhelming the user
		if i >= 9 {
			section += fmt.Sprintf("  ... and %d more\n", len(articles)-10)
			break
		}
	}

	return section
}

// formatArticleSummary formats a single article summary line.
func formatArticleSummary(index int, article articlev2.ReadableArticle) string {
	line := fmt.Sprintf("  %d. ID: %s, Name: %s, Template: %s, Orderable: %s",
		index,
		article.ArticleId,
		article.Name,
		article.Template.Name,
		article.Orderable,
	)

	if tagSummary := formatArticleTagsSummary(article.Tags); tagSummary != "" {
		line += tagSummary
	}

	if attrSummary := formatArticleAttributesSummary(article.Attributes); attrSummary != "" {
		line += attrSummary
	}

	return line + "\n"
}

// formatArticleTagsSummary formats the tags portion of an article summary.
func formatArticleTagsSummary(tags []articlev2.ArticleTag) string {
	if len(tags) == 0 {
		return ""
	}

	tagNames := make([]string, 0, len(tags))
	for _, tag := range tags {
		if tag.Name != nil {
			tagNames = append(tagNames, *tag.Name)
		}
	}

	if len(tagNames) == 0 {
		return ""
	}

	return fmt.Sprintf(", Tags: %v", tagNames)
}

// formatArticleAttributesSummary formats the attributes portion of an article summary.
func formatArticleAttributesSummary(attributes []articlev2.ArticleAttributes) string {
	if len(attributes) == 0 {
		return ""
	}

	attrPairs := make([]string, 0, len(attributes))
	for _, attr := range attributes {
		value := ""
		if attr.Value != nil {
			value = *attr.Value
		}
		attrPairs = append(attrPairs, fmt.Sprintf("%s=%s", attr.Key, value))
	}

	if len(attrPairs) == 0 {
		return ""
	}

	return fmt.Sprintf(", Attributes: {%s}", strings.Join(attrPairs, ", "))
}

// formatFilterSuggestionsSection formats the suggestions section of the error message.
func formatFilterSuggestionsSection() string {
	return "\nConsider adding more specific filters such as:\n" +
		"  - Additional tags (filter_tags)\n" +
		"  - Specific template names (filter_template)\n" +
		"  - More restrictive orderable status (filter_orderable)\n" +
		"  - Article attributes (filter_attributes)\n"
}
