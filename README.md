# summer

A Minimalist Mesh-Native Microservice Framework for Golang

## Core Concept

By using generics, `summer` allows you to use the basic `summer.Context`, or create your own `Context` type, without changing programming paradigm.

## Features

* Support `opentelemetry-go`
  * Support standard `OTEL_` environment variables
  * Using `zipkin` as default exporter
  * Support `TraceContext`, `Baggage` and `B3` propagation
  * Support `otelhttp` instrument
* Support `prometheus/promhttp`
  * Expose at `/debug/metrics`
* Support `Readiness Check`
  * Expose at `/debug/ready`
  * Component readiness registration with `App#Check()`
* Support `Liveness Check`
  * Expose at `/debug/alive`
  * Cascade `Liveness Check` failure from continuous `Readiness Check` failure
* Support `debug/pprof`
  * Expose at `/debug/pprof`
* Bind request data
  * Unmarshal `header`, `query`, `json body` and `form body` into any structure with `json` tag

## Setup Tracing

`OpenTelemetry` setup is left to end-user

Example:

```golang
package summer

import (
	"github.com/guoyk93/rg"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"net/http"
)

// BootTracing setup OpenTelemetry tracing best practice
func BootTracing() {
	otel.SetTracerProvider(sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(rg.Must(zipkin.New(""))),
	))
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
			b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader|b3.B3SingleHeader)),
		),
	)
	// re-initialize otelhttp.DefaultClient and http.DefaultClient
	otelhttp.DefaultClient = &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	http.DefaultClient = otelhttp.DefaultClient
}
```

## Donation

See https://guoyk.xyz/donation

## Credits

Guo Y.K., MIT License