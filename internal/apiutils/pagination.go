package apiutils

import (
	"context"
	"net/http"
	"strconv"
)

// FetchAllPages calls fetch repeatedly, incrementing the page number each time,
// until all pages have been retrieved. The total item count is determined from
// the x-pagination-totalcount response header returned by each call.
//
// fetch receives the page size and the 1-based page number for each request and
// must return the items for that page together with the raw HTTP response (so
// the pagination headers can be read).
func FetchAllPages[TItem any](ctx context.Context, pageSize int64, fetch func(ctx context.Context, limit, page int64) (*[]TItem, *http.Response, error)) ([]TItem, error) {
	all := make([]TItem, 0)

	for page := int64(1); ; page++ {
		items, resp, err := fetch(ctx, pageSize, page)
		if err != nil {
			return nil, err
		}

		if items != nil {
			all = append(all, *items...)
		}

		totalCount := paginationTotalCount(resp)
		if page*pageSize >= totalCount {
			break
		}
	}

	return all, nil
}

// paginationTotalCount reads the x-pagination-totalcount response header.
// Returns 0 if the header is absent or cannot be parsed.
func paginationTotalCount(resp *http.Response) int64 {
	if resp == nil {
		return 0
	}
	n, err := strconv.ParseInt(resp.Header.Get("x-pagination-totalcount"), 10, 64)
	if err != nil {
		return 0
	}
	return n
}
