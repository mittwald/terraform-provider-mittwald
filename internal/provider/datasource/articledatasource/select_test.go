package articledatasource

import (
	"testing"

	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/articlev2"
	. "github.com/onsi/gomega"
)

// floatPtr creates a float64 pointer.
func floatPtr(f float64) *float64 {
	return &f
}

// createArticleWithPrice creates an article with the specified ID and price.
func createArticleWithPrice(id string, price *float64) articlev2.ReadableArticle {
	return articlev2.ReadableArticle{
		ArticleId: id,
		Price:     price,
	}
}

func TestSelectByPrice(t *testing.T) {
	tests := []struct {
		name          string
		articles      []articlev2.ReadableArticle
		criterion     string
		expectedID    string
		expectedPrice float64
	}{
		{
			name: "lowest selects article with lowest price",
			articles: []articlev2.ReadableArticle{
				createArticleWithPrice("article-1", floatPtr(10.0)),
				createArticleWithPrice("article-2", floatPtr(5.0)),
				createArticleWithPrice("article-3", floatPtr(15.0)),
			},
			criterion:     "lowest",
			expectedID:    "article-2",
			expectedPrice: 5.0,
		},
		{
			name: "highest selects article with highest price",
			articles: []articlev2.ReadableArticle{
				createArticleWithPrice("article-1", floatPtr(10.0)),
				createArticleWithPrice("article-2", floatPtr(5.0)),
				createArticleWithPrice("article-3", floatPtr(15.0)),
			},
			criterion:     "highest",
			expectedID:    "article-3",
			expectedPrice: 15.0,
		},
		{
			name: "lowest with nil price treats as zero",
			articles: []articlev2.ReadableArticle{
				createArticleWithPrice("article-1", floatPtr(10.0)),
				createArticleWithPrice("article-2", nil),
				createArticleWithPrice("article-3", floatPtr(5.0)),
			},
			criterion:     "lowest",
			expectedID:    "article-2",
			expectedPrice: 0,
		},
		{
			name: "highest with nil price ignores nil",
			articles: []articlev2.ReadableArticle{
				createArticleWithPrice("article-1", floatPtr(10.0)),
				createArticleWithPrice("article-2", nil),
				createArticleWithPrice("article-3", floatPtr(15.0)),
			},
			criterion:     "highest",
			expectedID:    "article-3",
			expectedPrice: 15.0,
		},
		{
			name: "lowest with all same prices returns first",
			articles: []articlev2.ReadableArticle{
				createArticleWithPrice("article-1", floatPtr(10.0)),
				createArticleWithPrice("article-2", floatPtr(10.0)),
				createArticleWithPrice("article-3", floatPtr(10.0)),
			},
			criterion:     "lowest",
			expectedID:    "article-1",
			expectedPrice: 10.0,
		},
		{
			name: "highest with all same prices returns first",
			articles: []articlev2.ReadableArticle{
				createArticleWithPrice("article-1", floatPtr(10.0)),
				createArticleWithPrice("article-2", floatPtr(10.0)),
				createArticleWithPrice("article-3", floatPtr(10.0)),
			},
			criterion:     "highest",
			expectedID:    "article-1",
			expectedPrice: 10.0,
		},
		{
			name: "single article returns that article for lowest",
			articles: []articlev2.ReadableArticle{
				createArticleWithPrice("only-article", floatPtr(42.0)),
			},
			criterion:     "lowest",
			expectedID:    "only-article",
			expectedPrice: 42.0,
		},
		{
			name: "single article returns that article for highest",
			articles: []articlev2.ReadableArticle{
				createArticleWithPrice("only-article", floatPtr(42.0)),
			},
			criterion:     "highest",
			expectedID:    "only-article",
			expectedPrice: 42.0,
		},
		{
			name:          "empty articles returns empty article",
			articles:      []articlev2.ReadableArticle{},
			criterion:     "lowest",
			expectedID:    "",
			expectedPrice: 0,
		},
		{
			name: "lowest with negative prices",
			articles: []articlev2.ReadableArticle{
				createArticleWithPrice("article-1", floatPtr(-5.0)),
				createArticleWithPrice("article-2", floatPtr(-10.0)),
				createArticleWithPrice("article-3", floatPtr(5.0)),
			},
			criterion:     "lowest",
			expectedID:    "article-2",
			expectedPrice: -10.0,
		},
		{
			name: "highest with negative prices",
			articles: []articlev2.ReadableArticle{
				createArticleWithPrice("article-1", floatPtr(-5.0)),
				createArticleWithPrice("article-2", floatPtr(-10.0)),
				createArticleWithPrice("article-3", floatPtr(5.0)),
			},
			criterion:     "highest",
			expectedID:    "article-3",
			expectedPrice: 5.0,
		},
		{
			name: "lowest with decimal prices",
			articles: []articlev2.ReadableArticle{
				createArticleWithPrice("article-1", floatPtr(9.99)),
				createArticleWithPrice("article-2", floatPtr(9.98)),
				createArticleWithPrice("article-3", floatPtr(10.01)),
			},
			criterion:     "lowest",
			expectedID:    "article-2",
			expectedPrice: 9.98,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result := selectByPrice(tt.articles, tt.criterion)
			g.Expect(result.ArticleId).To(Equal(tt.expectedID))
			g.Expect(getArticlePrice(result)).To(Equal(tt.expectedPrice))
		})
	}
}

func TestGetArticlePrice(t *testing.T) {
	tests := []struct {
		name          string
		article       articlev2.ReadableArticle
		expectedPrice float64
	}{
		{
			name:          "returns price when set",
			article:       createArticleWithPrice("article-1", floatPtr(25.50)),
			expectedPrice: 25.50,
		},
		{
			name:          "returns zero when price is nil",
			article:       createArticleWithPrice("article-2", nil),
			expectedPrice: 0,
		},
		{
			name:          "returns zero price correctly",
			article:       createArticleWithPrice("article-3", floatPtr(0)),
			expectedPrice: 0,
		},
		{
			name:          "returns negative price correctly",
			article:       createArticleWithPrice("article-4", floatPtr(-15.0)),
			expectedPrice: -15.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)
			result := getArticlePrice(tt.article)
			g.Expect(result).To(Equal(tt.expectedPrice))
		})
	}
}
