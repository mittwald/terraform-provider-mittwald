package articledatasource

import (
	"testing"

	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/articlev2"
	. "github.com/onsi/gomega"
)

func TestFormatMultipleMatchesError(t *testing.T) {
	g := NewWithT(t)

	articles := []articlev2.ReadableArticle{
		{
			ArticleId: "article-1",
			Name:      "Premium Hosting",
			Template:  articlev2.ArticleTemplate{Name: "hosting"},
			Orderable: articlev2.ReadableArticleOrderableFull,
			Tags: []articlev2.ArticleTag{
				{Id: "tag-1", Name: strPtr("hosting")},
			},
			Attributes: []articlev2.ArticleAttributes{
				{Key: "cpu", Value: strPtr("4")},
			},
		},
		{
			ArticleId: "article-2",
			Name:      "Enterprise Hosting",
			Template:  articlev2.ArticleTemplate{Name: "hosting"},
			Orderable: articlev2.ReadableArticleOrderableFull,
		},
	}

	filterTags := []string{"hosting"}
	filterTemplate := []string{"hosting"}
	filterOrderable := []string{"full"}
	filterAttributes := map[string]string{"cpu": "4"}
	filterIDPattern := "article"

	result := formatMultipleMatchesError(articles, filterTags, filterTemplate, filterOrderable, filterAttributes, filterIDPattern)

	// Verify the error message contains key information
	g.Expect(result).To(ContainSubstring("Found 2 articles"))
	g.Expect(result).To(ContainSubstring("Applied filters:"))
	g.Expect(result).To(ContainSubstring("Tags: [hosting]"))
	g.Expect(result).To(ContainSubstring("Templates: [hosting]"))
	g.Expect(result).To(ContainSubstring("Orderable statuses: [full]"))
	g.Expect(result).To(ContainSubstring("cpu = 4"))
	g.Expect(result).To(ContainSubstring("ID pattern: article"))
	g.Expect(result).To(ContainSubstring("Matching articles:"))
	g.Expect(result).To(ContainSubstring("article-1"))
	g.Expect(result).To(ContainSubstring("article-2"))
	g.Expect(result).To(ContainSubstring("Consider adding more specific filters"))
}

func TestFormatMultipleMatchesErrorNoFilters(t *testing.T) {
	g := NewWithT(t)

	articles := []articlev2.ReadableArticle{
		{ArticleId: "article-1", Name: "Test", Template: articlev2.ArticleTemplate{Name: "test"}, Orderable: articlev2.ReadableArticleOrderableFull},
	}

	result := formatMultipleMatchesError(articles, nil, nil, nil, nil, "")

	g.Expect(result).To(ContainSubstring("No filters applied (matching all articles)"))
}

func TestFormatAppliedFiltersSection(t *testing.T) {
	tests := []struct {
		name             string
		filterTags       []string
		filterTemplate   []string
		filterOrderable  []string
		filterAttributes map[string]string
		filterIDPattern  string
		expectContains   []string
		expectNotContain []string
	}{
		{
			name:             "all filters applied",
			filterTags:       []string{"premium", "hosting"},
			filterTemplate:   []string{"hosting"},
			filterOrderable:  []string{"full"},
			filterAttributes: map[string]string{"cpu": "4", "ram": "8GB"},
			filterIDPattern:  "hosting-*",
			expectContains: []string{
				"Tags: [premium hosting]",
				"Templates: [hosting]",
				"Orderable statuses: [full]",
				"cpu = 4",
				"ram = 8GB",
				"ID pattern: hosting-*",
			},
			expectNotContain: []string{"No filters applied"},
		},
		{
			name:             "only tags filter",
			filterTags:       []string{"premium"},
			filterTemplate:   nil,
			filterOrderable:  nil,
			filterAttributes: nil,
			filterIDPattern:  "",
			expectContains:   []string{"Tags: [premium]"},
			expectNotContain: []string{"Templates:", "Orderable statuses:", "Attributes:", "ID pattern:"},
		},
		{
			name:             "no filters applied",
			filterTags:       nil,
			filterTemplate:   nil,
			filterOrderable:  nil,
			filterAttributes: nil,
			filterIDPattern:  "",
			expectContains:   []string{"No filters applied (matching all articles)"},
			expectNotContain: []string{"Tags:", "Templates:", "Orderable statuses:", "ID pattern:"},
		},
		{
			name:             "empty slices treated as no filters",
			filterTags:       []string{},
			filterTemplate:   []string{},
			filterOrderable:  []string{},
			filterAttributes: map[string]string{},
			filterIDPattern:  "",
			expectContains:   []string{"No filters applied (matching all articles)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := formatAppliedFiltersSection(tt.filterTags, tt.filterTemplate, tt.filterOrderable, tt.filterAttributes, tt.filterIDPattern)

			for _, expected := range tt.expectContains {
				g.Expect(result).To(ContainSubstring(expected), "expected to contain: %s", expected)
			}

			for _, notExpected := range tt.expectNotContain {
				g.Expect(result).NotTo(ContainSubstring(notExpected), "expected NOT to contain: %s", notExpected)
			}
		})
	}
}

func TestFormatMatchingArticlesList(t *testing.T) {
	g := NewWithT(t)

	articles := make([]articlev2.ReadableArticle, 15)
	for i := 0; i < 15; i++ {
		articles[i] = articlev2.ReadableArticle{
			ArticleId: "article-" + string(rune('a'+i)),
			Name:      "Article " + string(rune('A'+i)),
			Template:  articlev2.ArticleTemplate{Name: "template"},
			Orderable: articlev2.ReadableArticleOrderableFull,
		}
	}

	result := formatMatchingArticlesList(articles)

	// Should show first 10 articles
	g.Expect(result).To(ContainSubstring("article-a"))
	g.Expect(result).To(ContainSubstring("article-j"))

	// Should NOT show articles beyond 10
	g.Expect(result).NotTo(ContainSubstring("article-k"))
	g.Expect(result).NotTo(ContainSubstring("article-o"))

	// Should indicate there are more
	g.Expect(result).To(ContainSubstring("... and 5 more"))
}

func TestFormatMatchingArticlesListLessThan10(t *testing.T) {
	g := NewWithT(t)

	articles := []articlev2.ReadableArticle{
		{ArticleId: "article-1", Name: "Article 1", Template: articlev2.ArticleTemplate{Name: "template"}, Orderable: articlev2.ReadableArticleOrderableFull},
		{ArticleId: "article-2", Name: "Article 2", Template: articlev2.ArticleTemplate{Name: "template"}, Orderable: articlev2.ReadableArticleOrderableFull},
	}

	result := formatMatchingArticlesList(articles)

	g.Expect(result).To(ContainSubstring("article-1"))
	g.Expect(result).To(ContainSubstring("article-2"))
	g.Expect(result).NotTo(ContainSubstring("... and"))
}

func TestFormatArticleSummary(t *testing.T) {
	tests := []struct {
		name           string
		article        articlev2.ReadableArticle
		expectContains []string
	}{
		{
			name: "article with all fields",
			article: articlev2.ReadableArticle{
				ArticleId: "article-1",
				Name:      "Premium Hosting",
				Template:  articlev2.ArticleTemplate{Name: "hosting"},
				Orderable: articlev2.ReadableArticleOrderableFull,
				Tags: []articlev2.ArticleTag{
					{Id: "tag-1", Name: strPtr("premium")},
					{Id: "tag-2", Name: strPtr("hosting")},
				},
				Attributes: []articlev2.ArticleAttributes{
					{Key: "cpu", Value: strPtr("4")},
					{Key: "ram", Value: strPtr("8GB")},
				},
			},
			expectContains: []string{
				"ID: article-1",
				"Name: Premium Hosting",
				"Template: hosting",
				"Orderable: full",
				"Tags: [premium hosting]",
				"Attributes: {",
				"cpu=4",
				"ram=8GB",
			},
		},
		{
			name: "article without tags and attributes",
			article: articlev2.ReadableArticle{
				ArticleId: "article-2",
				Name:      "Basic Hosting",
				Template:  articlev2.ArticleTemplate{Name: "basic"},
				Orderable: articlev2.ReadableArticleOrderableDeprecated,
			},
			expectContains: []string{
				"ID: article-2",
				"Name: Basic Hosting",
				"Template: basic",
				"Orderable: deprecated",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			result := formatArticleSummary(1, tt.article)

			for _, expected := range tt.expectContains {
				g.Expect(result).To(ContainSubstring(expected), "expected to contain: %s", expected)
			}
		})
	}
}

func TestFormatArticleTagsSummary(t *testing.T) {
	tests := []struct {
		name     string
		tags     []articlev2.ArticleTag
		expected string
	}{
		{
			name:     "empty tags",
			tags:     []articlev2.ArticleTag{},
			expected: "",
		},
		{
			name:     "nil tags",
			tags:     nil,
			expected: "",
		},
		{
			name: "single tag",
			tags: []articlev2.ArticleTag{
				{Id: "tag-1", Name: strPtr("premium")},
			},
			expected: ", Tags: [premium]",
		},
		{
			name: "multiple tags",
			tags: []articlev2.ArticleTag{
				{Id: "tag-1", Name: strPtr("premium")},
				{Id: "tag-2", Name: strPtr("hosting")},
			},
			expected: ", Tags: [premium hosting]",
		},
		{
			name: "tags with nil names are filtered out",
			tags: []articlev2.ArticleTag{
				{Id: "tag-1", Name: strPtr("premium")},
				{Id: "tag-2", Name: nil},
			},
			expected: ", Tags: [premium]",
		},
		{
			name: "all tags have nil names",
			tags: []articlev2.ArticleTag{
				{Id: "tag-1", Name: nil},
				{Id: "tag-2", Name: nil},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result := formatArticleTagsSummary(tt.tags)
			g.Expect(result).To(Equal(tt.expected))
		})
	}
}

func TestFormatArticleAttributesSummary(t *testing.T) {
	tests := []struct {
		name           string
		attributes     []articlev2.ArticleAttributes
		expectEmpty    bool
		expectContains []string
	}{
		{
			name:        "empty attributes",
			attributes:  []articlev2.ArticleAttributes{},
			expectEmpty: true,
		},
		{
			name:        "nil attributes",
			attributes:  nil,
			expectEmpty: true,
		},
		{
			name: "single attribute",
			attributes: []articlev2.ArticleAttributes{
				{Key: "cpu", Value: strPtr("4")},
			},
			expectContains: []string{"Attributes: {", "cpu=4"},
		},
		{
			name: "multiple attributes",
			attributes: []articlev2.ArticleAttributes{
				{Key: "cpu", Value: strPtr("4")},
				{Key: "ram", Value: strPtr("8GB")},
			},
			expectContains: []string{"Attributes: {", "cpu=4", "ram=8GB"},
		},
		{
			name: "attribute with nil value",
			attributes: []articlev2.ArticleAttributes{
				{Key: "optional", Value: nil},
			},
			expectContains: []string{"optional="},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result := formatArticleAttributesSummary(tt.attributes)

			if tt.expectEmpty {
				g.Expect(result).To(BeEmpty())
			} else {
				for _, expected := range tt.expectContains {
					g.Expect(result).To(ContainSubstring(expected))
				}
			}
		})
	}
}

func TestFormatFilterSuggestionsSection(t *testing.T) {
	g := NewWithT(t)

	result := formatFilterSuggestionsSection()

	g.Expect(result).To(ContainSubstring("Consider adding more specific filters"))
	g.Expect(result).To(ContainSubstring("filter_tags"))
	g.Expect(result).To(ContainSubstring("filter_template"))
	g.Expect(result).To(ContainSubstring("filter_orderable"))
	g.Expect(result).To(ContainSubstring("filter_attributes"))
}
