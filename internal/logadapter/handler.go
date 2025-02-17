package logadapter

import (
	"context"
	"log/slog"
	"sync"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ slog.Handler = &TFLHandler{}

// TFLHandler is a custom slog.Handler implementation using tflog.
type TFLHandler struct {
	mu     sync.Mutex
	attrs  []slog.Attr
	groups []string
}

// Enabled checks if logging should be enabled for the given level.
func (h *TFLHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// Enable logging for all levels (modify as needed)
	return true
}

// Handle processes a log record and maps it to tflog.
func (h *TFLHandler) Handle(ctx context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Resolve attributes
	fields := make(map[string]interface{})
	for _, attr := range h.attrs {
		fields[attr.Key] = attr.Value.Any()
	}
	record.Attrs(func(a slog.Attr) bool {
		fields[a.Key] = a.Value.Any()
		return true
	})

	// Apply groups
	if len(h.groups) > 0 {
		groupedFields := make(map[string]interface{})
		for k, v := range fields {
			groupedFields[h.groups[len(h.groups)-1]+"."+k] = v
		}
		fields = groupedFields
	}

	// Determine appropriate tflog function
	switch record.Level {
	case slog.LevelDebug:
		tflog.Debug(ctx, record.Message, fields)
	case slog.LevelInfo:
		tflog.Info(ctx, record.Message, fields)
	case slog.LevelWarn:
		tflog.Warn(ctx, record.Message, fields)
	case slog.LevelError:
		tflog.Error(ctx, record.Message, fields)
	default:
		tflog.Trace(ctx, record.Message, fields)
	}

	return nil
}

// WithAttrs returns a new handler with the provided attributes.
func (h *TFLHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.mu.Lock()
	defer h.mu.Unlock()

	newHandler := &TFLHandler{
		attrs:  append(h.attrs, attrs...),
		groups: h.groups,
	}
	return newHandler
}

// WithGroup returns a new handler with the given group.
func (h *TFLHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	newHandler := &TFLHandler{
		attrs:  h.attrs,
		groups: append(h.groups, name),
	}
	return newHandler
}
