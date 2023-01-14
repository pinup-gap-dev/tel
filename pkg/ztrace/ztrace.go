package ztrace

import (
	"github.com/pkg/errors"
	"github.com/tel-io/tel/v2/otlplog/logskd"
	"github.com/tel-io/tel/v2/pkg/attrencoder"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
)

var (
	ErrNotRecording = errors.New("span is not recording")
)

type Core struct {
	trace.Span
	enc *attrencoder.AtrEncoder
	lvl zapcore.Level
}

func New(lvl zapcore.Level, span trace.Span) zapcore.Core {
	return &Core{lvl: lvl, Span: span, enc: attrencoder.NewAttr()}
}

func (c *Core) clone() *Core {
	return &Core{
		Span: c.Span,
		enc:  c.enc.Clone(),
		lvl:  c.lvl,
	}
}

func (c *Core) With(fields []zapcore.Field) zapcore.Core {
	clone := c.clone()
	//addFields(clone.enc, fields)

	for i := range fields {
		if fields[i].Key == logskd.SpanKey {
			continue
		}

		fields[i].AddTo(c.enc)
	}

	return clone
}

func (c *Core) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	if !c.Span.IsRecording() {
		return errors.WithStack(ErrNotRecording)
	}

	e, err := c.enc.EncodeEntry(entry, fields)
	if err != nil {
		return errors.WithStack(err)
	}

	c.Span.AddEvent(entry.Message)
	c.Span.SetAttributes(e...)

	if entry.Level == zapcore.ErrorLevel {
		c.Span.SetStatus(codes.Error, "error_mark")
	}

	return nil
}

func (c *Core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}

	return ce
}

func (c Core) Sync() error { return nil }

func (c Core) Enabled(lvl zapcore.Level) bool { return lvl >= c.lvl }
