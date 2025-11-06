package articledatasource

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	mittwaldv2 "github.com/mittwald/api-client-go/mittwaldv2/generated/clients"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/clients/articleclientv2"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/articlev2"
	"github.com/mittwald/terraform-provider-mittwald/internal/provider/providerutil"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ArticleDataSource{}

func New() datasource.DataSource {
	return &ArticleDataSource{}
}

// DataSource defines the data source implementation.
type ArticleDataSource struct {
	client mittwaldv2.Client
}

func (d *ArticleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_article"
}

func (d *ArticleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `A data source that selects different articles.

This data source should typically be used in conjunction with the ` + "`mittwald_server`" + ` or ` + "`mittwald_project`" + `
resources to select the respective hosting plan.

**Important:** The filters must match exactly one article. If no articles match, or if multiple articles match the
specified criteria, the data source will return an error with detailed information about the matching articles to
help you refine your filters.`,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the selected article.",
			},
			"filter_tags": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "A list of tags to filter articles by. Only articles containing all specified tags will be considered.",
			},
			"filter_template": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "A list of templates to filter articles by.",
			},
			"filter_orderable": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "A list of orderable statuses to filter articles by.",
			},
			"filter_attributes": schema.MapAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "A map of attributes to filter articles by. Only articles with matching attribute key-value pairs will be considered. All specified attributes must match.",
			},
			"orderable": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The orderable status of the selected article.",
			},
			"price": schema.Float64Attribute{
				Computed:            true,
				MarkdownDescription: "The price of the selected article.",
			},
			"attributes": schema.MapAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "The attributes of the selected article as key-value pairs.",
			},
			"tags": schema.ListAttribute{
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"name": types.StringType,
						"id":   types.StringType,
					},
				},
				Computed:            true,
				MarkdownDescription: "The tags associated with the selected article.",
			},
		},
	}
}

func (d *ArticleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = providerutil.ClientFromProviderData(req.ProviderData, &resp.Diagnostics)
}

func (d *ArticleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract filter values from the model
	filters := d.extractFilterValues(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch and filter articles
	articles := d.fetchAndFilterArticles(ctx, filters, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that exactly one article matched
	matchedArticle := d.validateSingleMatch(articles, filters, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Map the article to the model
	resp.Diagnostics.Append(data.FromAPIModel(matchedArticle)...)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// articleFilters holds all filter values extracted from the Terraform configuration
type articleFilters struct {
	tags       []string
	templates  []string
	orderable  []string
	attributes map[string]string
}

// extractFilterValues extracts all filter values from the data model
func (d *ArticleDataSource) extractFilterValues(ctx context.Context, data *DataSourceModel, diags *diag.Diagnostics) articleFilters {
	filters := articleFilters{
		attributes: make(map[string]string),
	}

	if !data.FilterTags.IsNull() {
		diags.Append(data.FilterTags.ElementsAs(ctx, &filters.tags, false)...)
	}

	if !data.FilterTemplate.IsNull() {
		diags.Append(data.FilterTemplate.ElementsAs(ctx, &filters.templates, false)...)
	}

	if !data.FilterOrderable.IsNull() {
		diags.Append(data.FilterOrderable.ElementsAs(ctx, &filters.orderable, false)...)
	}

	if !data.FilterAttributes.IsNull() {
		diags.Append(data.FilterAttributes.ElementsAs(ctx, &filters.attributes, false)...)
	}

	return filters
}

// fetchAndFilterArticles fetches articles from the API and applies client-side filtering
func (d *ArticleDataSource) fetchAndFilterArticles(ctx context.Context, filters articleFilters, diags *diag.Diagnostics) *[]articlev2.ReadableArticle {
	// Build the API request
	listReq := d.buildListArticlesRequest(filters)

	// Fetch articles from API
	articleClient := d.client.Article()
	articles, _, err := articleClient.ListArticles(ctx, listReq)
	if err != nil {
		diags.AddError("Failed to list articles", err.Error())
		return nil
	}

	if articles == nil || len(*articles) == 0 {
		diags.AddError(
			"No matching article found",
			"No article matched the specified filter criteria. Please check your filter values and try again.",
		)
		return nil
	}

	// Apply client-side attribute filtering if specified
	if len(filters.attributes) > 0 {
		articles = d.applyAttributeFiltering(*articles, filters.attributes, diags)
	}

	return articles
}

// buildListArticlesRequest builds the API request from filter values
func (d *ArticleDataSource) buildListArticlesRequest(filters articleFilters) articleclientv2.ListArticlesRequest {
	listReq := articleclientv2.ListArticlesRequest{}

	if len(filters.tags) > 0 {
		listReq.Tags = filters.tags
	}

	if len(filters.templates) > 0 {
		listReq.TemplateNames = filters.templates
	}

	if len(filters.orderable) > 0 {
		orderableItems := make([]articleclientv2.ListArticlesRequestQueryOrderableItem, 0, len(filters.orderable))
		for _, orderable := range filters.orderable {
			orderableItems = append(orderableItems, articleclientv2.ListArticlesRequestQueryOrderableItem(orderable))
		}
		listReq.Orderable = orderableItems
	}

	return listReq
}

// applyAttributeFiltering filters articles based on attribute key-value pairs
func (d *ArticleDataSource) applyAttributeFiltering(articles []articlev2.ReadableArticle, filterAttributes map[string]string, diags *diag.Diagnostics) *[]articlev2.ReadableArticle {
	filteredArticles := make([]articlev2.ReadableArticle, 0)
	for _, article := range articles {
		if matchesAttributeFilters(article, filterAttributes) {
			filteredArticles = append(filteredArticles, article)
		}
	}

	if len(filteredArticles) == 0 {
		diags.AddError(
			"No matching article found",
			"No article matched the specified filter criteria. Please check your filter values and try again.",
		)
		return nil
	}

	return &filteredArticles
}

// validateSingleMatch ensures exactly one article matched the filters
func (d *ArticleDataSource) validateSingleMatch(articles *[]articlev2.ReadableArticle, filters articleFilters, diags *diag.Diagnostics) articlev2.ReadableArticle {
	if articles == nil {
		return articlev2.ReadableArticle{}
	}

	if len(*articles) > 1 {
		diags.AddError(
			"Multiple articles matched",
			formatMultipleMatchesError(*articles, filters.tags, filters.templates, filters.orderable, filters.attributes),
		)
		return articlev2.ReadableArticle{}
	}

	return (*articles)[0]
}

// matchesAttributeFilters checks if an article matches the given attribute filters
func matchesAttributeFilters(article articlev2.ReadableArticle, filterAttributes map[string]string) bool {
	// Build a map of the article's attributes
	articleAttrs := make(map[string]string)
	for _, attr := range article.Attributes {
		if attr.Value != nil {
			articleAttrs[attr.Key] = *attr.Value
		} else {
			articleAttrs[attr.Key] = ""
		}
	}

	// Check if all filter attributes match
	for key, value := range filterAttributes {
		articleValue, exists := articleAttrs[key]
		if !exists || articleValue != value {
			return false
		}
	}

	return true
}

// formatMultipleMatchesError creates a comprehensive error message when multiple articles match the filter criteria
func formatMultipleMatchesError(articles []articlev2.ReadableArticle, filterTags, filterTemplate, filterOrderable []string, filterAttributes map[string]string) string {
	// Build a detailed error message with filter information
	errorDetail := fmt.Sprintf("Found %d articles matching the specified filters. Please refine your filters to match exactly one article.\n\n", len(articles))

	// Show which filters were applied
	errorDetail += "Applied filters:\n"
	if len(filterTags) > 0 {
		errorDetail += fmt.Sprintf("  - Tags: %v\n", filterTags)
	}
	if len(filterTemplate) > 0 {
		errorDetail += fmt.Sprintf("  - Templates: %v\n", filterTemplate)
	}
	if len(filterOrderable) > 0 {
		errorDetail += fmt.Sprintf("  - Orderable statuses: %v\n", filterOrderable)
	}
	if len(filterAttributes) > 0 {
		errorDetail += "  - Attributes:\n"
		for key, value := range filterAttributes {
			errorDetail += fmt.Sprintf("      %s = %s\n", key, value)
		}
	}
	if len(filterTags) == 0 && len(filterTemplate) == 0 && len(filterOrderable) == 0 && len(filterAttributes) == 0 {
		errorDetail += "  - No filters applied (matching all articles)\n"
	}

	// List the matching articles
	errorDetail += "\nMatching articles:\n"
	for i, article := range articles {
		errorDetail += fmt.Sprintf("  %d. ID: %s, Name: %s, Template: %s, Orderable: %s",
			i+1,
			article.ArticleId,
			article.Name,
			article.Template,
			article.Orderable,
		)

		// Add tags if present
		if len(article.Tags) > 0 {
			tagNames := make([]string, 0, len(article.Tags))
			for _, tag := range article.Tags {
				if tag.Name != nil {
					tagNames = append(tagNames, *tag.Name)
				}
			}
			if len(tagNames) > 0 {
				errorDetail += fmt.Sprintf(", Tags: %v", tagNames)
			}
		}

		// Add attributes if present
		if len(article.Attributes) > 0 {
			attrPairs := make([]string, 0, len(article.Attributes))
			for _, attr := range article.Attributes {
				value := ""
				if attr.Value != nil {
					value = *attr.Value
				}
				attrPairs = append(attrPairs, fmt.Sprintf("%s=%s", attr.Key, value))
			}
			if len(attrPairs) > 0 {
				errorDetail += fmt.Sprintf(", Attributes: {%s}", fmt.Sprintf("%v", attrPairs))
			}
		}
		errorDetail += "\n"

		// Limit to showing first 10 articles to avoid overwhelming the user
		if i >= 9 {
			errorDetail += fmt.Sprintf("  ... and %d more\n", len(articles)-10)
			break
		}
	}

	errorDetail += "\nConsider adding more specific filters such as:\n"
	errorDetail += "  - Additional tags (filter_tags)\n"
	errorDetail += "  - Specific template names (filter_template)\n"
	errorDetail += "  - More restrictive orderable status (filter_orderable)\n"
	errorDetail += "  - Article attributes (filter_attributes)\n"

	return errorDetail
}
