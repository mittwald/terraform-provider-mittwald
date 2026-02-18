package airesource

import (
	"context"
	"fmt"
	"strconv"

	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/articleclientv2"
)

func (r *ResourceModel) QueryArticleFeatures(ctx context.Context, client mittwaldv2.Client) (int64, int64, error) {
	monthlyTokens := int64(0)
	requestsPerMinute := int64(0)

	articleID := r.ArticleID.ValueString()

	articleRequest := articleclientv2.GetArticleRequest{ArticleID: articleID}
	article, _, err := client.Article().GetArticle(ctx, articleRequest)
	if err != nil {
		return 0, 0, fmt.Errorf("error while retrieving article %s: %w", articleID, err)
	}

	for _, attr := range article.Attributes {
		if attr.Value == nil {
			continue
		}

		if attr.Key == "monthlyTokens" {
			monthlyTokens, err = strconv.ParseInt(*attr.Value, 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("error while parsing monthlyTokens: %w", err)
			}
		}

		if attr.Key == "requestsPerMinute" {
			requestsPerMinute, err = strconv.ParseInt(*attr.Value, 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("error while parsing requestsPerMinute: %w", err)
			}
		}
	}

	return monthlyTokens, requestsPerMinute, nil
}
