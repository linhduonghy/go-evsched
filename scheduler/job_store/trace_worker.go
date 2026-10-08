package job_store

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type TraceWorker struct {
	river.WorkerDefaults[GenericJobArgs]

	baseLogger *slog.Logger
	tracer     trace.Tracer
	spanName   string
	inner      *Worker
}

func NewTraceWorker(logger *slog.Logger, tracer trace.Tracer, inner *Worker) *TraceWorker {
	return &TraceWorker{
		baseLogger: logger,
		tracer:     tracer,
		spanName:   "river-job-worker",
		inner:      inner,
	}
}

func (w *TraceWorker) Work(ctx context.Context, job *river.Job[GenericJobArgs]) error {

	ctx, span := w.tracer.Start(ctx, w.spanName, trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	span.SetAttributes(
		attribute.String("job_id", job.Args.ManagedJobID.String()),
		attribute.Int64("river_job_id", job.JobRow.ID),
	)

	err := w.inner.Work(ctx, job)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "job execution failed")
		return err
	}

	return nil

}
