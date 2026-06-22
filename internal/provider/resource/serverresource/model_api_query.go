package serverresource

import (
	"context"
	"fmt"

	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/articleclientv2"
)

// QueryArticleMachineType reads the machine type for the configured article from
// the article's `machine_type` attribute.
func (r *ResourceModel) QueryArticleMachineType(ctx context.Context, client mittwaldv2.Client) (string, error) {
	articleID := r.ArticleID.ValueString()

	articleRequest := articleclientv2.GetArticleRequest{ArticleID: articleID}
	article, _, err := client.Article().GetArticle(ctx, articleRequest)
	if err != nil {
		return "", fmt.Errorf("error while retrieving article %s: %w", articleID, err)
	}

	for _, attr := range article.Attributes {
		if attr.Value == nil {
			continue
		}

		if attr.Key == "machine_type" {
			return *attr.Value, nil
		}
	}

	return "", fmt.Errorf("article %s does not have a 'machine_type' attribute", articleID)
}
