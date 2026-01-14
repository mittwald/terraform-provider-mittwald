package articledatasource

import (
	"testing"

	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/articlev2"
	. "github.com/onsi/gomega"
)

// strPtr creates a string pointer.
func strPtr(s string) *string {
	return &s
}

// createArticleWithTags creates an article with the specified ID and tags.
func createArticleWithTags(id string, tagNames ...string) articlev2.ReadableArticle {
	tags := make([]articlev2.ArticleTag, 0, len(tagNames))
	for i, name := range tagNames {
		tags = append(tags, articlev2.ArticleTag{
			Id:   "tag-" + string(rune('a'+i)),
			Name: strPtr(name),
		})
	}
	return articlev2.ReadableArticle{
		ArticleId: id,
		Tags:      tags,
	}
}

// createArticleWithAttributes creates an article with the specified ID and attributes.
func createArticleWithAttributes(id string, attrs map[string]string) articlev2.ReadableArticle {
	attributes := make([]articlev2.ArticleAttributes, 0, len(attrs))
	for key, value := range attrs {
		v := value
		attributes = append(attributes, articlev2.ArticleAttributes{
			Key:   key,
			Value: &v,
		})
	}
	return articlev2.ReadableArticle{
		ArticleId:  id,
		Attributes: attributes,
	}
}

func TestMatchesTagFilters(t *testing.T) {
	tests := []struct {
		name        string
		article     articlev2.ReadableArticle
		filterTags  []string
		expectMatch bool
	}{
		{
			name:        "empty filter matches any article",
			article:     createArticleWithTags("hosting-basic", "hosting", "premium"),
			filterTags:  []string{},
			expectMatch: true,
		},
		{
			name:        "single tag filter matches article with that tag",
			article:     createArticleWithTags("hosting-premium", "hosting", "premium"),
			filterTags:  []string{"hosting"},
			expectMatch: true,
		},
		{
			name:        "single tag filter does not match article without that tag",
			article:     createArticleWithTags("hosting-standard", "hosting", "premium"),
			filterTags:  []string{"enterprise"},
			expectMatch: false,
		},
		{
			name:        "multiple tag filters match article with all tags (AND logic)",
			article:     createArticleWithTags("hosting-ssd", "hosting", "premium", "ssd"),
			filterTags:  []string{"hosting", "premium"},
			expectMatch: true,
		},
		{
			name:        "multiple tag filters do not match article missing one tag",
			article:     createArticleWithTags("hosting-hdd", "hosting", "premium"),
			filterTags:  []string{"hosting", "enterprise"},
			expectMatch: false,
		},
		{
			name:        "article with no tags does not match any filter",
			article:     createArticleWithTags("no-tags"),
			filterTags:  []string{"hosting"},
			expectMatch: false,
		},
		{
			name:        "empty filter matches article with no tags",
			article:     createArticleWithTags("empty-tags"),
			filterTags:  []string{},
			expectMatch: true,
		},
		{
			name:        "tag filter is case sensitive",
			article:     createArticleWithTags("case-sensitive", "Hosting"),
			filterTags:  []string{"hosting"},
			expectMatch: false,
		},
		{
			name: "article with nil tag name is ignored",
			article: articlev2.ReadableArticle{
				ArticleId: "article-1",
				Tags: []articlev2.ArticleTag{
					{Id: "tag-a", Name: strPtr("hosting")},
					{Id: "tag-b", Name: nil},
				},
			},
			filterTags:  []string{"hosting"},
			expectMatch: true,
		},
		{
			name:        "filter all tags present in article",
			article:     createArticleWithTags("all-tags", "a", "b", "c"),
			filterTags:  []string{"a", "b", "c"},
			expectMatch: true,
		},
		{
			name:        "filter with more tags than article has",
			article:     createArticleWithTags("few-tags", "a", "b"),
			filterTags:  []string{"a", "b", "c", "d"},
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result := matchesTagFilters(tt.article, tt.filterTags)
			g.Expect(result).To(Equal(tt.expectMatch))
		})
	}
}

func TestMatchesAttributeFilters(t *testing.T) {
	tests := []struct {
		name             string
		article          articlev2.ReadableArticle
		filterAttributes map[string]string
		expectMatch      bool
	}{
		{
			name:             "empty filter matches any article",
			article:          createArticleWithAttributes("server-small", map[string]string{"cpu": "4", "ram": "8GB"}),
			filterAttributes: map[string]string{},
			expectMatch:      true,
		},
		{
			name:             "single attribute filter matches article with matching attribute",
			article:          createArticleWithAttributes("server-medium", map[string]string{"cpu": "4", "ram": "8GB"}),
			filterAttributes: map[string]string{"cpu": "4"},
			expectMatch:      true,
		},
		{
			name:             "single attribute filter does not match article with different value",
			article:          createArticleWithAttributes("server-large", map[string]string{"cpu": "4", "ram": "8GB"}),
			filterAttributes: map[string]string{"cpu": "8"},
			expectMatch:      false,
		},
		{
			name:             "single attribute filter does not match article without that attribute",
			article:          createArticleWithAttributes("server-basic", map[string]string{"cpu": "4", "ram": "8GB"}),
			filterAttributes: map[string]string{"storage": "100GB"},
			expectMatch:      false,
		},
		{
			name:             "multiple attribute filters match article with all matching attributes",
			article:          createArticleWithAttributes("server-full", map[string]string{"cpu": "4", "ram": "8GB", "storage": "100GB"}),
			filterAttributes: map[string]string{"cpu": "4", "ram": "8GB"},
			expectMatch:      true,
		},
		{
			name:             "multiple attribute filters do not match if one attribute differs",
			article:          createArticleWithAttributes("server-mismatch", map[string]string{"cpu": "4", "ram": "8GB"}),
			filterAttributes: map[string]string{"cpu": "4", "ram": "16GB"},
			expectMatch:      false,
		},
		{
			name:             "article with no attributes does not match any filter",
			article:          createArticleWithAttributes("server-empty", map[string]string{}),
			filterAttributes: map[string]string{"cpu": "4"},
			expectMatch:      false,
		},
		{
			name:             "empty filter matches article with no attributes",
			article:          createArticleWithAttributes("server-noattr", map[string]string{}),
			filterAttributes: map[string]string{},
			expectMatch:      true,
		},
		{
			name:             "attribute filter is case sensitive for keys",
			article:          createArticleWithAttributes("server-case-key", map[string]string{"CPU": "4"}),
			filterAttributes: map[string]string{"cpu": "4"},
			expectMatch:      false,
		},
		{
			name:             "attribute filter is case sensitive for values",
			article:          createArticleWithAttributes("server-case-val", map[string]string{"tier": "Premium"}),
			filterAttributes: map[string]string{"tier": "premium"},
			expectMatch:      false,
		},
		{
			name: "attribute with nil value matches empty string filter",
			article: articlev2.ReadableArticle{
				ArticleId: "article-1",
				Attributes: []articlev2.ArticleAttributes{
					{Key: "optional", Value: nil},
				},
			},
			filterAttributes: map[string]string{"optional": ""},
			expectMatch:      true,
		},
		{
			name: "attribute with nil value does not match non-empty filter",
			article: articlev2.ReadableArticle{
				ArticleId: "article-1",
				Attributes: []articlev2.ArticleAttributes{
					{Key: "optional", Value: nil},
				},
			},
			filterAttributes: map[string]string{"optional": "value"},
			expectMatch:      false,
		},
		{
			name:             "filter matches empty string value",
			article:          createArticleWithAttributes("server-emptyval", map[string]string{"empty": ""}),
			filterAttributes: map[string]string{"empty": ""},
			expectMatch:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result := matchesAttributeFilters(tt.article, tt.filterAttributes)
			g.Expect(result).To(Equal(tt.expectMatch))
		})
	}
}

func TestMatchesIDPattern(t *testing.T) {
	tests := []struct {
		name        string
		article     articlev2.ReadableArticle
		pattern     string
		expectMatch bool
	}{
		{
			name:        "empty pattern matches empty article ID",
			article:     articlev2.ReadableArticle{ArticleId: ""},
			pattern:     "",
			expectMatch: true,
		},
		{
			name:        "empty pattern does not match non-empty article ID",
			article:     articlev2.ReadableArticle{ArticleId: "hosting-premium-2023"},
			pattern:     "",
			expectMatch: false,
		},
		{
			name:        "exact match pattern matches article ID",
			article:     articlev2.ReadableArticle{ArticleId: "hosting-premium"},
			pattern:     "hosting-premium",
			expectMatch: true,
		},
		{
			name:        "exact pattern does not match longer article ID",
			article:     articlev2.ReadableArticle{ArticleId: "hosting-premium-2023"},
			pattern:     "hosting-premium",
			expectMatch: false,
		},
		{
			name:        "substring pattern without wildcards does not match",
			article:     articlev2.ReadableArticle{ArticleId: "hosting-premium-2023"},
			pattern:     "premium",
			expectMatch: false,
		},
		{
			name:        "wildcard pattern matches prefix",
			article:     articlev2.ReadableArticle{ArticleId: "hosting-premium-2023"},
			pattern:     "hosting*",
			expectMatch: true,
		},
		{
			name:        "wildcard pattern with full prefix matches",
			article:     articlev2.ReadableArticle{ArticleId: "hosting-premium-2023"},
			pattern:     "hosting-premium*",
			expectMatch: true,
		},
		{
			name:        "wildcard pattern does not match different prefix",
			article:     articlev2.ReadableArticle{ArticleId: "hosting-premium-2023"},
			pattern:     "enterprise*",
			expectMatch: false,
		},
		{
			name:        "wildcard pattern exact prefix match",
			article:     articlev2.ReadableArticle{ArticleId: "hosting-premium-2023"},
			pattern:     "hosting-premium-2023*",
			expectMatch: true,
		},
		{
			name:        "suffix wildcard pattern matches",
			article:     articlev2.ReadableArticle{ArticleId: "hosting-premium-2023"},
			pattern:     "*2023",
			expectMatch: true,
		},
		{
			name:        "suffix wildcard pattern does not match wrong suffix",
			article:     articlev2.ReadableArticle{ArticleId: "hosting-premium-2023"},
			pattern:     "*2024",
			expectMatch: false,
		},
		{
			name:        "wildcard in middle matches",
			article:     articlev2.ReadableArticle{ArticleId: "hosting-premium-2023"},
			pattern:     "host*2023",
			expectMatch: true,
		},
		{
			name:        "wildcard in middle matches complex pattern",
			article:     articlev2.ReadableArticle{ArticleId: "hosting-premium-2023"},
			pattern:     "host*pre*23",
			expectMatch: true,
		},
		{
			name:        "wildcard in middle does not match if parts missing",
			article:     articlev2.ReadableArticle{ArticleId: "hosting-premium-2023"},
			pattern:     "host*enterprise*2023",
			expectMatch: false,
		},
		{
			name:        "pattern is case sensitive",
			article:     articlev2.ReadableArticle{ArticleId: "Hosting-Premium"},
			pattern:     "hosting-premium",
			expectMatch: false,
		},
		{
			name:        "wildcard pattern is case sensitive",
			article:     articlev2.ReadableArticle{ArticleId: "Hosting-Premium"},
			pattern:     "hosting*",
			expectMatch: false,
		},
		{
			name:        "just wildcard matches all",
			article:     articlev2.ReadableArticle{ArticleId: "any-article-id"},
			pattern:     "*",
			expectMatch: true,
		},
		{
			name:        "double wildcard matches",
			article:     articlev2.ReadableArticle{ArticleId: "hosting-premium-2023"},
			pattern:     "**",
			expectMatch: true,
		},
		{
			name:        "question mark matches single character",
			article:     articlev2.ReadableArticle{ArticleId: "abc"},
			pattern:     "a?c",
			expectMatch: true,
		},
		{
			name:        "question mark does not match multiple characters",
			article:     articlev2.ReadableArticle{ArticleId: "abbc"},
			pattern:     "a?c",
			expectMatch: false,
		},
		{
			name:        "character class matches",
			article:     articlev2.ReadableArticle{ArticleId: "hosting-1"},
			pattern:     "hosting-[123]",
			expectMatch: true,
		},
		{
			name:        "character class does not match outside range",
			article:     articlev2.ReadableArticle{ArticleId: "hosting-5"},
			pattern:     "hosting-[123]",
			expectMatch: false,
		},
		{
			name:        "empty article ID does not match non-empty pattern",
			article:     articlev2.ReadableArticle{ArticleId: ""},
			pattern:     "hosting",
			expectMatch: false,
		},
		{
			name:        "empty article ID matches wildcard only",
			article:     articlev2.ReadableArticle{ArticleId: ""},
			pattern:     "*",
			expectMatch: true,
		},
		{
			name:        "invalid pattern returns false",
			article:     articlev2.ReadableArticle{ArticleId: "hosting"},
			pattern:     "[",
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result := matchesIDPattern(tt.article, tt.pattern)
			g.Expect(result).To(Equal(tt.expectMatch))
		})
	}
}
