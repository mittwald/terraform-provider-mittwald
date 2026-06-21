# Refactoring hand-off: shared order/contract handling

This is a **hand-off note for a future session**. It documents the duplication
that now exists between the order-backed resources and proposes a shared helper
package. **Nothing here has been implemented** — the `mittwald_server` resource
was deliberately written by mirroring `mittwald_ai` so the two can be compared
side by side before extracting common code.

## Affected resources

- `internal/provider/resource/airesource/` (`mittwald_ai`)
- `internal/provider/resource/serverresource/` (`mittwald_server`)

Both are *order-backed*: creation goes through `contractclientv2.CreateOrder`,
plan changes through `contractclientv2.CreateTariffChange`, and deletion through
`contractclientv2.TerminateContract`. Any future order-backed product
(project hosting, mail archive, licenses, …) will repeat the same shape.

## Duplicated concerns

### 1. Order creation
Both resources build a `contractclientv2.CreateOrderRequest` (differing only in
the `OrderType` + `Alternative*Order` payload), call `CreateOrder`, and wrap both
steps in `providerutil.Try[...]`. See:
- `airesource/resource.go` `createNewContract`
- `serverresource/resource.go` `Create`

### 2. Tariff change
Identical structure (`CreateTariffChange` + `Try`), differing only in the
`Alternative*TariffChange` payload. See both resources' `changePlan`.

### 3. Contract termination / cancellation
- `TerminateContract` on delete (both, with `IgnoreNotFound()`).
- `CancelContractTermination` (currently only `airesource.adoptExistingContract`,
  but generally useful).

### 4. Article attribute lookup
Both query an article and pull typed values out of `article.Attributes`:
- `airesource/model_api_query.go` `QueryArticleFeatures` → `monthlyTokens`,
  `requestsPerMinute` (int64)
- `serverresource/model_api_query.go` `QueryArticleMachineType` → `machine_type`
  (string)

These should collapse into a single generic attribute getter.

## Proposed shared package

Create `internal/provider/resource/common/order` (or `internal/orderutil`) with
roughly the following surface. Each helper takes the `mittwaldv2.Client` and a
`*diag.Diagnostics` (or returns `(T, error)`) to keep call sites terse.

```go
// CreateOrder issues an order and returns the order ID.
func CreateOrder(ctx, client, body contractclientv2.CreateOrderRequestBody, diags) string

// CreateTariffChange issues a tariff change.
func CreateTariffChange(ctx, client, body contractclientv2.CreateTariffChangeRequestBody, diags)

// TerminateContract terminates a contract, ignoring "not found".
func TerminateContract(ctx, client, contractID string, diags)

// CancelContractTermination cancels a pending termination.
func CancelContractTermination(ctx, client, contractID string, diags)

// GetArticleAttribute returns the raw string value of an article attribute,
// or ("", false) when absent. Callers parse/convert as needed.
func GetArticleAttribute(ctx, client, articleID, key string) (string, bool, error)
```

The `serverresource`-specific **order → server resolution** logic
(`resolveServerFromOrder`: poll `GetOrder` → match `ListContracts` by
`BaseItem.ItemId`/`OrderId` → `AggregateReference.Id`) is a candidate for a more
general `ResolveAggregateFromOrder(ctx, client, orderID, customerID)` helper if
another aggregate-backed (as opposed to customer-singleton) order resource is
added later. The AI resource does not need it because AI hosting is a
one-contract-per-customer product looked up directly via
`GetDetailOfContractByAIHosting`.

## Suggested order of work

1. Extract `GetArticleAttribute` first (smallest, no behavioral change) and
   rewrite both `model_api_query.go` files on top of it.
2. Extract the `CreateOrder` / `CreateTariffChange` / `TerminateContract`
   wrappers.
3. Re-evaluate whether a generic order-lifecycle abstraction is worth it once a
   third order-backed resource appears (avoid premature generalization).

Do not implement during the current `mittwald_server` work.
