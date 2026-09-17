// Package telemetry 按官方文档初始化 OpenTelemetry SDK：
// https://opentelemetry.io/docs/languages/go/getting-started/
package telemetry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// Setup 初始化全局 TracerProvider 与 W3C TraceContext 传播器，
// 返回的 shutdown 函数必须在进程退出前调用，以 flush 未导出的 span。
//
// 导出端点等参数遵循官方标准环境变量约定：
//
//	OTEL_EXPORTER_OTLP_ENDPOINT      OTLP endpoint，如 localhost:4317
//	OTEL_EXPORTER_OTLP_HEADERS       附加请求头
//	OTEL_RESOURCE_ATTRIBUTES         追加资源属性
func Setup(ctx context.Context, serviceName, serviceVersion string) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown 合并所有组件的关闭函数，任一失败也继续关闭其余组件
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	handleErr := func(inErr error) {
		joinErr := errors.Join(inErr, shutdown(ctx))
		err = joinErr
	}

	// OTLP gRPC exporter 默认读取 OTEL_EXPORTER_OTLP_ENDPOINT 等环境变量
	traceExporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		handleErr(fmt.Errorf("create OTLP trace exporter: %w", err))
		return
	}
	shutdownFuncs = append(shutdownFuncs, traceExporter.Shutdown)

	res, err := resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		))
	if err != nil {
		handleErr(fmt.Errorf("create resource: %w", err))
		return
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter,
			// 默认 5s，这里调短便于观察
			sdktrace.WithBatchTimeout(time.Second)),
		sdktrace.WithResource(res),
	)
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return shutdown, nil
}
