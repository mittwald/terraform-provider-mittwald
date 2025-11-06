package articledatasource

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mittwald/api-client-go/mittwaldv2/generated/schemas/articlev2"
)

// DataSourceModel describes the data source data model.
type DataSourceModel struct {
	FilterTags       types.List    `tfsdk:"filter_tags"`
	FilterTemplate   types.List    `tfsdk:"filter_template"`
	FilterOrderable  types.List    `tfsdk:"filter_orderable"`
	FilterAttributes types.Map     `tfsdk:"filter_attributes"`
	ID               types.String  `tfsdk:"id"`
	Orderable        types.String  `tfsdk:"orderable"`
	Price            types.Float64 `tfsdk:"price"`
	Attributes       types.Map     `tfsdk:"attributes"`
	Tags             types.List    `tfsdk:"tags"`
}

type TagModel struct {
	Name types.String `tfsdk:"name"`
	ID   types.String `tfsdk:"id"`
}

// FromAPIModel maps from the API model to the Terraform model
func (m *DataSourceModel) FromAPIModel(article articlev2.ReadableArticle) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ID = types.StringValue(article.ArticleId)
	m.Orderable = types.StringValue(string(article.Orderable))

	// Handle price - use 0.0 if not present
	price := 0.0
	if article.Price != nil {
		price = *article.Price
	}
	m.Price = types.Float64Value(price)

	// Map tags
	tagObjects := make([]attr.Value, 0, len(article.Tags))
	for _, tag := range article.Tags {
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

	// Map attributes to a string map
	attributeMap := make(map[string]attr.Value)
	for _, attribute := range article.Attributes {
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
