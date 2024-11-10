# otlprobe

## Introduction

This  application works as a service which accepts OTLP signals via HTTP/gRPC.
Received data can be presented in a convenient way in the interactive mode, where you can filter, browse or examine attributes.

![Image](demo.gif)

## Installation

To build and install application from the source code call the following command:

```
go build
go install
```

## Getting started

To start application in the interactive mode use the following command:

```
otlprobe
```

by default it accepts connections via gRPC/HTTP on ports 4317/4318.

Now you can configure your collector or your application to send signals to `otlprobe`.

```
exporters:
  otlp:
    endpoint: localhost:4317
    tls:
      insecure: true
      insecure_skip_verify: true
    compression: none
```

## Features / Roadmap

* interactive and non-interactive mode
* read all types of signals
* support grpc/http protocol (insecure)
* TODO: support secure grpc/http
* TODO: graphs with metrics in interactive mode
* TODO: docker image
