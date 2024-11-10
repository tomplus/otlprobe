package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pmetric/pmetricotlp"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"google.golang.org/grpc"
)

type Server struct {
	grpcPort int
	httpPort int
	ch       chan *Signal

	serverGRPC *grpc.Server
	serverHTTP *http.Server
}

func newServer(grpcPort int, httpPort int, ch chan *Signal) *Server {
	s := Server{
		grpcPort: grpcPort,
		httpPort: httpPort,
		ch:       ch,
	}
	return &s
}

type metricsServer struct {
	pmetricotlp.UnimplementedGRPCServer
	server *Server
}

type logServer struct {
	plogotlp.UnimplementedGRPCServer
	server *Server
}

type traceServer struct {
	ptraceotlp.UnimplementedGRPCServer
	server *Server
}

func (ms metricsServer) Export(_ context.Context, request pmetricotlp.ExportRequest) (pmetricotlp.ExportResponse, error) {
	m := request.Metrics()
	ms.server.processMetrics(&m)
	return pmetricotlp.NewExportResponse(), nil
}

func (ls logServer) Export(_ context.Context, request plogotlp.ExportRequest) (plogotlp.ExportResponse, error) {
	l := request.Logs()
	ls.server.processLogs(&l)
	return plogotlp.NewExportResponse(), nil
}

func (ls traceServer) Export(_ context.Context, request ptraceotlp.ExportRequest) (ptraceotlp.ExportResponse, error) {
	l := request.Traces()
	ls.server.processTraces(&l)
	return ptraceotlp.NewExportResponse(), nil
}

func (server *Server) start() {
	if server.grpcPort > 0 {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", server.grpcPort))
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		server.serverGRPC = grpc.NewServer()
		pmetricotlp.RegisterGRPCServer(server.serverGRPC, &metricsServer{server: server})
		plogotlp.RegisterGRPCServer(server.serverGRPC, &logServer{server: server})
		ptraceotlp.RegisterGRPCServer(server.serverGRPC, &traceServer{server: server})
		go func() {
			server.serverGRPC.Serve(lis)
		}()
	}
	if server.httpPort > 0 {
		mux := http.NewServeMux()
		mux.HandleFunc("/v1/metrics", server.httpMetricHandler)
		mux.HandleFunc("/v1/logs", server.httpLogHandler)
		mux.HandleFunc("/v1/traces", server.httpTraceHandler)
		server.serverHTTP = &http.Server{Addr: fmt.Sprintf(":%v", server.httpPort), Handler: mux}
		err := server.serverHTTP.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			panic("http server closed\n")
		} else if err != nil {
			panic(fmt.Sprintf("error listening http: %s\n", err))
		}
	}
}

func (server *Server) httpMetricHandler(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		resp.Header().Set("Content-Type", "text/plain")
		resp.WriteHeader(http.StatusMethodNotAllowed)
		resp.Write([]byte("Method not allowed"))
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Printf("could not read body: %s\n", err)
	}
	preq := pmetricotlp.NewExportRequest()
	err = preq.UnmarshalProto(body)
	if err != nil {
		fmt.Println(preq, err)
	}
	ms := preq.Metrics()
	server.processMetrics(&ms)
	presp := pmetricotlp.NewExportResponse()
	pb, err := presp.MarshalJSON()
	if err != nil {
		fmt.Println(err)
	}
	resp.WriteHeader(http.StatusOK)
	resp.Write(pb)
}

func (server *Server) processMetrics(ms *pmetric.Metrics) {
	rms := ms.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		resProps := newPropsContainer("Resource")
		resProps.addMap(rm.Resource().Attributes(), "Attributes")
		resProps.addUInt32("DroppedAttributesCount", rm.Resource().DroppedAttributesCount())
		sms := rm.ScopeMetrics()
		for j := 0; j < sms.Len(); j++ {
			sm := sms.At(j)
			scopeProps := newPropsContainer("Scope")
			scopeProps.addString("Name", sm.Scope().Name())
			ms := sm.Metrics()
			for k := 0; k < ms.Len(); k++ {
				m := ms.At(k)
				props := newPropsContainer("Metric")
				props.addString("Type", m.Type().String())
				props.addString("Name", m.Name())
				props.addString("Unit", m.Unit())
				props.addString("Description", m.Description())

				switch m.Type() {
				case pmetric.MetricTypeGauge:
					dp := m.Gauge().DataPoints()
					for l := 0; l < dp.Len(); l++ {
						dpProps := newPropsContainer("DataPoint")
						dpp := dp.At(l)
						dpProps.addBool("Flags.NoRecordedValue", dpp.Flags().NoRecordedValue())
						dpProps.addUInt32("Exemplars", uint32(dpp.Exemplars().Len()))
						dpProps.addMap(dpp.Attributes(), "Attributes")
						valstr := "N/A"
						if dpp.ValueType() == pmetric.NumberDataPointValueTypeInt {
							valstr = fmt.Sprintf("%v", dpp.IntValue())
						} else if dpp.ValueType() == pmetric.NumberDataPointValueTypeDouble {
							valstr = fmt.Sprintf("%v", dpp.DoubleValue())
						}
						dpProps.addString("Value", valstr)
						dpProps.addString("ValueType", dpp.ValueType().String())
						s := Signal{
							time:       dpp.Timestamp(),
							summary:    fmt.Sprintf("%v=%v", m.Name(), valstr),
							properties: []Properties{dpProps, props, scopeProps, resProps},
							kind:       METRIC,
						}
						server.ch <- &s
					}
				case pmetric.MetricTypeSum:
					props.addString("AggregationTemporality", m.Sum().AggregationTemporality().String())
					props.addBool("IsMonotonic", m.Sum().IsMonotonic())
					dp := m.Sum().DataPoints()
					for l := 0; l < dp.Len(); l++ {
						dpProps := newPropsContainer("DataPoint")
						dpp := dp.At(l)
						dpProps.addBool("Flags.NoRecordedValue", dpp.Flags().NoRecordedValue())
						dpProps.addMap(dpp.Attributes(), "Attributes")
						valstr := "N/A"
						if dpp.ValueType() == pmetric.NumberDataPointValueTypeInt {
							valstr = fmt.Sprintf("%v", dpp.IntValue())
						} else if dpp.ValueType() == pmetric.NumberDataPointValueTypeDouble {
							valstr = fmt.Sprintf("%v", dpp.DoubleValue())
						}
						dpProps.addString("Value", valstr)
						dpProps.addString("ValueType", dpp.ValueType().String())
						s := Signal{
							time:       dpp.Timestamp(),
							summary:    fmt.Sprintf("%v=%v [%v]", m.Name(), valstr, dpProps),
							properties: []Properties{dpProps, props, scopeProps, resProps},
							kind:       METRIC,
						}
						server.ch <- &s
					}
				case pmetric.MetricTypeHistogram:
					dp := m.Histogram().DataPoints()
					for l := 0; l < dp.Len(); l++ {
						dpp := dp.At(l)
						// probably we can use buckets as label with value
						dpp.BucketCounts().At(0)
						dpp.ExplicitBounds().At(0)
						props.addBool("Flags.NoRecordedValue", dpp.Flags().NoRecordedValue())
						s := Signal{
							time:       dpp.Timestamp(),
							summary:    fmt.Sprintf("%v=%v", m.Name(), "HISTOGRAM"),
							properties: []Properties{props, scopeProps, resProps},
							kind:       METRIC,
						}
						server.ch <- &s
					}
				case pmetric.MetricTypeExponentialHistogram:
					m.ExponentialHistogram().DataPoints()
				case pmetric.MetricTypeSummary: // Summary (Legacy)
					m.Summary().DataPoints()
				case pmetric.MetricTypeEmpty:
					// ?
				}

			}
		}
	}
}

func (server *Server) httpLogHandler(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		resp.Header().Set("Content-Type", "text/plain")
		resp.WriteHeader(http.StatusMethodNotAllowed)
		resp.Write([]byte("Method not allowed"))
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Printf("could not read body: %s\n", err)
	}
	preq := plogotlp.NewExportRequest()
	err = preq.UnmarshalProto(body)
	if err != nil {
		fmt.Println(preq, err)
	}
	ls := preq.Logs()
	server.processLogs(&ls)
	presp := pmetricotlp.NewExportResponse()
	pb, err := presp.MarshalJSON()
	if err != nil {
		fmt.Println(err)
	}
	resp.WriteHeader(http.StatusOK)
	resp.Write(pb)
}

func (server *Server) processLogs(ms *plog.Logs) {
	rls := ms.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		rl := rls.At(i)
		resProps := newPropsContainer("Resource")
		resProps.addMap(rl.Resource().Attributes(), "Attributes")
		resProps.addUInt32("DroppedAttributesCount", rl.Resource().DroppedAttributesCount())
		sls := rl.ScopeLogs()
		for j := 0; j < sls.Len(); j++ {
			sl := sls.At(j)
			scopeProps := newPropsContainer("Scope")
			scopeProps.addString("Name", sl.Scope().Name())
			rs := sl.LogRecords()
			for k := 0; k < rs.Len(); k++ {
				r := rs.At(k)
				props := newPropsContainer("Record")
				props.addMap(r.Attributes(), "Attributes")
				props.addTimestamp("Timestamp", r.Timestamp())
				props.addTimestamp("ObservedTimestamp", r.ObservedTimestamp())
				props.addString("TraceId", r.TraceID().String())
				props.addString("SpanId", r.SpanID().String())
				props.addBool("Flags.IsSampled", r.Flags().IsSampled())
				props.addString("SeverityText", r.SeverityText())
				props.addString("SeverityNumber", r.SeverityNumber().String())
				props.addString("Body", r.Body().AsString())
				s := Signal{
					kind:       LOG,
					time:       r.Timestamp(),
					summary:    fmt.Sprintf("%v: %v", r.SeverityText(), r.Body().AsString()),
					properties: []Properties{props, scopeProps, resProps},
				}
				server.ch <- &s
			}
		}
	}
}

func (server *Server) httpTraceHandler(resp http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		resp.Header().Set("Content-Type", "text/plain")
		resp.WriteHeader(http.StatusMethodNotAllowed)
		resp.Write([]byte("Method not allowed"))
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		fmt.Printf("could not read body: %s\n", err)
	}
	preq := ptraceotlp.NewExportRequest()
	err = preq.UnmarshalProto(body)
	if err != nil {
		fmt.Println(preq, err)
	}
	ls := preq.Traces()
	server.processTraces(&ls)
	presp := ptraceotlp.NewExportResponse()
	pb, err := presp.MarshalJSON()
	if err != nil {
		fmt.Println(err)
	}
	resp.WriteHeader(http.StatusOK)
	resp.Write(pb)
}

func (server *Server) processTraces(ts *ptrace.Traces) {
	rss := ts.ResourceSpans()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		resProps := newPropsContainer("Resource")
		resProps.addMap(rs.Resource().Attributes(), "Attributes")
		resProps.addUInt32("DroppedAttributesCount", rs.Resource().DroppedAttributesCount())
		sss := rs.ScopeSpans()
		for j := 0; j < sss.Len(); j++ {
			ss := sss.At(j)
			scopeProps := newPropsContainer("Scope")
			scopeProps.addString("Name", ss.Scope().Name())
			rs := ss.Spans()
			for k := 0; k < rs.Len(); k++ {
				sp := rs.At(k)
				spanProps := newPropsContainer("Span")
				spanProps.addMap(sp.Attributes(), "Attributes")
				s := Signal{
					time:       sp.StartTimestamp(),
					summary:    fmt.Sprintf("[%v], %v, %v, %v, %v, %v, %d", sp.Kind().String(), sp.Status().Message(), sp.Name(), sp.TraceID(), sp.SpanID(), sp.ParentSpanID(), sp.Events().Len()),
					properties: []Properties{spanProps, scopeProps, resProps},
					kind:       TRACE,
				}
				server.ch <- &s
			}
		}
	}
}
