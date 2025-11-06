package articledatasource

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/articlev2"
)

// DataSourceModel describes the data source data model.
type DataSourceModel struct {
	Filter     types.Object  `tfsdk:"filter"`
	ID         types.String  `tfsdk:"id"`
	Orderable  types.String  `tfsdk:"orderable"`
	Price      types.Float64 `tfsdk:"price"`
	Attributes types.Map     `tfsdk:"attributes"`
	Tags       types.List    `tfsdk:"tags"`
}

type DataSourceFilterModel struct {
	Tags       types.List `tfsdk:"tags"`
	Template   types.List `tfsdk:"template"`
	Orderable  types.List `tfsdk:"orderable"`
	Attributes types.Map  `tfsdk:"attributes"`
}

type TagModel struct {
	Name types.String `tfsdk:"name"`
	ID   types.String `tfsdk:"id"`
}

// FromAPIModel maps from the API model to the Terraform model
func (m *DataSourceModel) FromAPIModel(article articlev2.ReadableArticle) diag.Diagnostics {
	var diags diag.Diagnostics

	m.mapBasicFields(article)

	diags.Append(m.mapTags(article.Tags)...)
	diags.Append(m.mapAttributes(article.Attributes)...)

	return diags
}

// mapBasicFields maps the basic article fields (ID, orderable, price)
func (m *DataSourceModel) mapBasicFields(article articlev2.ReadableArticle) {
	m.ID = types.StringValue(article.ArticleId)
	m.Orderable = types.StringValue(string(article.Orderable))

	price := 0.0
	if article.Price != nil {
		price = *article.Price
	}
	m.Price = types.Float64Value(price)
}

// mapTags maps article tags to the Terraform list format
func (m *DataSourceModel) mapTags(tags []articlev2.ArticleTag) diag.Diagnostics {
	var diags diag.Diagnostics

	tagObjects := make([]attr.Value, 0, len(tags))
	for _, tag := range tags {
		tagName := ""
		if tag.Name != nil {
			tagName = *tag.Name
		}

		tagObj, d := types.ObjectValue(
			map[string]attr.Type{
				"name": types.StringType,
				"id":   types.StringType,
			},
			map[string]attr.Value{
				"name": types.StringValue(tagName),
				"id":   types.StringValue(tag.Id),
			},
		)
		diags.Append(d...)
		tagObjects = append(tagObjects, tagObj)
	}

	tagList, d := types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name": types.StringType,
				"id":   types.StringType,
			},
		},
		tagObjects,
	)
	diags.Append(d...)
	m.Tags = tagList

	return diags
}

// mapAttributes maps article attributes to the Terraform map format
func (m *DataSourceModel) mapAttributes(attributes []articlev2.ArticleAttributes) diag.Diagnostics {
	var diags diag.Diagnostics

	attributeMap := make(map[string]attr.Value)
	for _, attribute := range attributes {
		value := ""
		if attribute.Value != nil {
			value = *attribute.Value
		}
		attributeMap[attribute.Key] = types.StringValue(value)
	}

	attributesMapValue, d := types.MapValue(types.StringType, attributeMap)
	diags.Append(d...)
	m.Attributes = attributesMapValue

	return diags
}
