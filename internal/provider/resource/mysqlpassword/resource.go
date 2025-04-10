package mysqlpassword

import (
	"context"
	crypto_rand "crypto/rand"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"math/big"
	math_rand "math/rand"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ ephemeral.EphemeralResource = &Resource{}

func New() ephemeral.EphemeralResource {
	return &Resource{}
}

type Resource struct {
}

type MySQLPasswordModel struct {
	Length   types.Int32  `tfsdk:"length"`
	Password types.String `tfsdk:"password"`
}

func (m *MySQLPasswordModel) LengthOrDefault() int {
	if m.Length.IsNull() {
		return 16
	}

	return int(m.Length.ValueInt32())
}

func (r *Resource) Metadata(_ context.Context, request ephemeral.MetadataRequest, response *ephemeral.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_mysql_password"
}

func (r *Resource) Schema(_ context.Context, _ ephemeral.SchemaRequest, response *ephemeral.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Generate a random MySQL password compliant with the MySQL password policy.",
		Attributes: map[string]schema.Attribute{
			"length": schema.Int32Attribute{
				Description: "The desired length of the password. The default is 16.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "The generated password.",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func (r *Resource) Open(ctx context.Context, request ephemeral.OpenRequest, response *ephemeral.OpenResponse) {
	model := MySQLPasswordModel{}

	response.Diagnostics.Append(request.Config.Get(ctx, &model)...)

	password, err := generatePassword(model.LengthOrDefault())
	if err != nil {
		response.Diagnostics.AddAttributeError(path.Root("password"), "Error generating password", "Unable to generate password: "+err.Error())
		return
	}

	model.Password = types.StringValue(password)

	response.Result.Set(ctx, &model)
}

func generatePassword(length int) (string, error) {
	// Ensure the minimum length is 8
	if length < 8 {
		length = 8
	}

	// Character sets
	lowercase := "abcdefghijklmnopqrstuvwxyz"
	uppercase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits := "0123456789"
	specialChars := "#!~%^*_+-=?{}()<>|.,;"

	// Ensure the password does not start with any of these characters
	disallowedStart := "-_;"

	// Combine all character sets
	allChars := lowercase + uppercase + digits + specialChars

	// Function to generate a random character from a given set
	randomChar := func(set string) (byte, error) {
		n := len(set)
		index, err := crypto_rand.Int(crypto_rand.Reader, big.NewInt(int64(n)))
		if err != nil {
			return 0, err
		}
		return set[index.Int64()], nil
	}

	// Generate the initial required characters
	password := []byte{}

	// Ensure at least one of each required character type is included
	char, err := randomChar(lowercase)
	if err != nil {
		return "", err
	}
	password = append(password, char)

	char, err = randomChar(uppercase)
	if err != nil {
		return "", err
	}
	password = append(password, char)

	char, err = randomChar(digits)
	if err != nil {
		return "", err
	}
	password = append(password, char)

	char, err = randomChar(specialChars)
	if err != nil {
		return "", err
	}
	password = append(password, char)

	// Fill up the remaining characters randomly from all available characters
	for len(password) < length {
		char, err := randomChar(allChars)
		if err != nil {
			return "", err
		}
		password = append(password, char)
	}

	// Ensure the password doesn't start with disallowed characters
	for strings.ContainsAny(string(password[0]), disallowedStart) {
		password[0], _ = randomChar(allChars)
	}

	// Shuffle the password but preserve the first character
	shuffledPassword := shuffleBytes(password[1:])
	// Combine the first character with the shuffled rest
	finalPassword := append([]byte{password[0]}, shuffledPassword...)

	return string(finalPassword), nil
}

// Helper function to shuffle a slice of bytes
func shuffleBytes(input []byte) []byte {
	n := len(input)
	output := make([]byte, n)
	perm := math_rand.Perm(n)

	for i, v := range perm {
		output[v] = input[i]
	}

	return output
}
