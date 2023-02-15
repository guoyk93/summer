# summer

A Minimalist Mesh-Native Microservice Framework for Golang

## Features

* `OpenTelemetry` supported
* `PromHTTP` supported
  * Exposed at `/debug/metrics`
* `Readiness Check` supported
  * Exposed at `/debug/ready`
  * Component readiness registration with `App#Check()`
* `Liveness Check` supported
  * Exposed at `/debug/alive`
  * Cascade `Liveness Check` failure from continuous `Readiness Check` failure
* `pprof` supported
  * Exposed at `/debug/pprof`
* Request data binding
  * Unmarshal header, query, json and form data into any structure with `json` tag

## Donation

See https://guoyk.xyz/donation

## Credits

Guo Y.K., MIT License