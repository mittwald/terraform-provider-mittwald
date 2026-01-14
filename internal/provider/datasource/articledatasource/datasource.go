package articledatasource

import (
	"context"
	"path"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
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
			"filter": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Filter criteria to select the article. All specified criteria must match (AND logic).",
				Attributes: map[string]schema.Attribute{
					"tags": schema.ListAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						MarkdownDescription: "A list of tags to filter articles by. Articles must have ALL specified tags (AND logic).",
					},
					"template": schema.ListAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						MarkdownDescription: "A list of templates to filter articles by.",
					},
					"orderable": schema.ListAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						MarkdownDescription: "A list of orderable statuses to filter articles by.",
					},
					"attributes": schema.MapAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						MarkdownDescription: "A map of attributes to filter articles by. Only articles with matching attribute key-value pairs will be considered. All specified attributes must match.",
					},
					"id": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "A pattern to match against article IDs. Only articles whose IDs contain this pattern will be considered. Use `*` as a wildcard to match any sequence of characters.",
					},
				},
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
	var filter DataSourceFilterModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	resp.Diagnostics.Append(data.Filter.As(ctx, &filter, basetypes.ObjectAsOptions{})...)
	if resp.Diagnostics.HasError() {
		return
	}

	filters := d.extractFilterValues(ctx, &filter, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	articles := d.fetchAndFilterArticles(ctx, filters, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	matchedArticle := d.validateSingleMatch(articles, filters, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(data.FromAPIModel(matchedArticle)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// articleFilters holds all filter values extracted from the Terraform configuration.
type articleFilters struct {
	tags       []string
	templates  []string
	orderable  []string
	attributes map[string]string
	idPattern  string
}

// extractFilterValues extracts all filter values from the data model.
func (d *ArticleDataSource) extractFilterValues(ctx context.Context, data *DataSourceFilterModel, diags *diag.Diagnostics) articleFilters {
	filters := articleFilters{
		attributes: make(map[string]string),
	}

	if !data.Tags.IsNull() {
		diags.Append(data.Tags.ElementsAs(ctx, &filters.tags, false)...)
	}

	if !data.Template.IsNull() {
		diags.Append(data.Template.ElementsAs(ctx, &filters.templates, false)...)
	}

	if !data.Orderable.IsNull() {
		diags.Append(data.Orderable.ElementsAs(ctx, &filters.orderable, false)...)
	}

	if !data.Attributes.IsNull() {
		diags.Append(data.Attributes.ElementsAs(ctx, &filters.attributes, false)...)
	}

	if !data.ID.IsNull() {
		filters.idPattern = data.ID.ValueString()
	}

	return filters
}

// fetchAndFilterArticles fetches articles from the API and applies client-side filtering.
func (d *ArticleDataSource) fetchAndFilterArticles(ctx context.Context, filters articleFilters, diags *diag.Diagnostics) *[]articlev2.ReadableArticle {
	var articles *[]articlev2.ReadableArticle
	var err error

	articleClient := d.client.Article()

	if filters.idPattern != "" && !strings.Contains(filters.idPattern, "*") {
		getReq := articleclientv2.GetArticleRequest{ArticleID: filters.idPattern}
		article, _, err := articleClient.GetArticle(ctx, getReq)
		if err != nil {
			diags.AddError("Failed to get article by ID", err.Error())
			return nil
		}

		articles = &[]articlev2.ReadableArticle{*article}
	} else {
		listReq := d.buildListArticlesRequest(filters)
		articles, _, err = articleClient.ListArticles(ctx, listReq)
		if err != nil {
			diags.AddError("Failed to list articles", err.Error())
			return nil
		}
	}

	if articles == nil || len(*articles) == 0 {
		diags.AddError(
			"No matching article found",
			"No article matched the specified filter criteria. Please check your filter values and try again.",
		)
		return nil
	}

	if len(filters.tags) > 0 {
		articles = d.applyTagFiltering(*articles, filters.tags, diags)
		if articles == nil {
			return nil
		}
	}

	if len(filters.attributes) > 0 {
		articles = d.applyAttributeFiltering(*articles, filters.attributes, diags)
		if articles == nil {
			return nil
		}
	}

	if filters.idPattern != "" {
		articles = d.applyIDPatternFiltering(*articles, filters.idPattern, diags)
	}

	return articles
}

// buildListArticlesRequest builds the API request from filter values.
func (d *ArticleDataSource) buildListArticlesRequest(filters articleFilters) articleclientv2.ListArticlesRequest {
	listReq := articleclientv2.ListArticlesRequest{}

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

// applyTagFiltering filters articles based on tags using AND logic.
func (d *ArticleDataSource) applyTagFiltering(articles []articlev2.ReadableArticle, filterTags []string, diags *diag.Diagnostics) *[]articlev2.ReadableArticle {
	filteredArticles := make([]articlev2.ReadableArticle, 0)
	for _, article := range articles {
		if matchesTagFilters(article, filterTags) {
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

// applyAttributeFiltering filters articles based on attribute key-value pairs.
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

// applyIDPatternFiltering filters articles based on ID pattern matching.
func (d *ArticleDataSource) applyIDPatternFiltering(articles []articlev2.ReadableArticle, pattern string, diags *diag.Diagnostics) *[]articlev2.ReadableArticle {
	filteredArticles := make([]articlev2.ReadableArticle, 0)
	for _, article := range articles {
		if matchesIDPattern(article, pattern) {
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

// validateSingleMatch ensures exactly one article matched the filters.
func (d *ArticleDataSource) validateSingleMatch(articles *[]articlev2.ReadableArticle, filters articleFilters, diags *diag.Diagnostics) articlev2.ReadableArticle {
	if articles == nil {
		return articlev2.ReadableArticle{}
	}

	if len(*articles) > 1 {
		diags.AddError(
			"Multiple articles matched",
			formatMultipleMatchesError(*articles, filters.tags, filters.templates, filters.orderable, filters.attributes, filters.idPattern),
		)
		return articlev2.ReadableArticle{}
	}

	return (*articles)[0]
}

// matchesTagFilters checks if an article has all the specified tags (AND logic).
func matchesTagFilters(article articlev2.ReadableArticle, filterTags []string) bool {
	articleTagNames := make(map[string]bool)
	for _, tag := range article.Tags {
		if tag.Name != nil {
			articleTagNames[*tag.Name] = true
		}
	}

	for _, requiredTag := range filterTags {
		if !articleTagNames[requiredTag] {
			return false
		}
	}

	return true
}

// matchesAttributeFilters checks if an article matches the given attribute filters.
func matchesAttributeFilters(article articlev2.ReadableArticle, filterAttributes map[string]string) bool {
	articleAttrs := make(map[string]string)
	for _, a := range article.Attributes {
		if a.Value != nil {
			articleAttrs[a.Key] = *a.Value
		} else {
			articleAttrs[a.Key] = ""
		}
	}

	for key, value := range filterAttributes {
		articleValue, exists := articleAttrs[key]
		if !exists || articleValue != value {
			return false
		}
	}

	return true
}

// matchesIDPattern checks if an article's ID matches the given pattern.
// The pattern supports glob-style wildcard matching where '*' matches any sequence of characters.
func matchesIDPattern(article articlev2.ReadableArticle, pattern string) bool {
	matched, err := path.Match(pattern, article.ArticleId)
	if err != nil {
		return false
	}
	return matched
}
