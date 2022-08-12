package hachibi

import (
	"bytes"
	"context"
	"net/http"
	"time"
)

type Writer struct {
	w          http.ResponseWriter
	body       bytes.Buffer
	statusCode int
}

func newWriter(w http.ResponseWriter) *Writer {
	var statusCode int = 0
	return &Writer{
		w:          w,
		body:       bytes.Buffer{},
		statusCode: statusCode,
	}
}

func (w *Writer) Header() http.Header {
	return w.w.Header()
}

func (w *Writer) Write(i []byte) (int, error) {
	w.body.Write(i)
	return w.w.Write(i)
}

func (w *Writer) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.w.WriteHeader(statusCode)
}

const (
	KeyErrorCtxMiddleware = "middleware-error"
)

func AddErrorInMiddlewareCtx(ctx context.Context, err error) context.Context {
	errCollections := make([]error, 0)
	errCollections = append(errCollections, err)

	errFromCtx := ctx.Value(KeyErrorCtxMiddleware)
	if errFromCtx != nil {
		errs, ok := errFromCtx.([]error)
		if ok {
			errCollections = append(errCollections, errs...)
		}
	}

	return context.WithValue(ctx, KeyErrorCtxMiddleware, err)
}

func getErrorInMiddlewareCtx(ctx context.Context) []error {
	err, ok := ctx.Value(KeyErrorCtxMiddleware).([]error)
	if !ok {
		return nil
	}

	return err
}

type Middleware struct {
	processor    Processor
	preProcessor PreProcessor
	errorHandler ErrorHandler

	eventName string
}

type MiddlewareOpt func(*Middleware)

func MiddlewareWithProcessor(processor Processor) MiddlewareOpt {
	return func(middleware *Middleware) {
		middleware.processor = processor
	}
}

func MiddlewareWithErrorHandler(errorHandler ErrorHandler) MiddlewareOpt {
	return func(middleware *Middleware) {
		middleware.errorHandler = errorHandler
	}
}

func MiddlewareWithPreProcessor(preProcessor PreProcessor) MiddlewareOpt {
	return func(middleware *Middleware) {
		middleware.preProcessor = preProcessor
	}
}

func NewMiddleware(opts ...MiddlewareOpt) *Middleware {
	m := Middleware{}
	for _, opt := range opts {
		opt(&m)
	}

	return &m
}

func (m Middleware) SetPreProcessor(preProcessor PreProcessor) *Middleware {
	newM := m
	newM.preProcessor = preProcessor
	return &newM
}

func (m Middleware) SetProcessor(processor Processor) *Middleware {
	newM := m
	newM.processor = processor

	return &newM
}

func (m Middleware) SetErrorHandler(handler ErrorHandler) *Middleware {
	newM := m
	newM.errorHandler = handler

	return &newM
}

func (m Middleware) SetEventName(event string) *Middleware {
	newM := m
	newM.eventName = event

	return &newM
}

func (m Middleware) Middleware(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		timeStart := time.Now().Local()
		writerClone := newWriter(writer)

		defer func() {

			ctx := request.Context()
			duration := time.Since(timeStart)
			httpData := &HttpData{
				Duration:   duration.Nanoseconds(),
				URL:        request.URL.String(),
				Method:     request.Method,
				StatusCode: writerClone.statusCode,
				Event:      m.eventName,
			}

			if err := getErrorInMiddlewareCtx(ctx); err != nil {
				httpData.Error = append(httpData.Error, err...)
			}

			err := httpData.extractRequest(request)
			if err != nil {
				httpData.Error = append(httpData.Error, err)
			}

			httpData.Response = Response{Payload{
				Header: writerClone.w.Header().Clone(),
				Body:   writerClone.body.Bytes(),
			}}

			if m.preProcessor != nil {
				err := m.preProcessor.PreProcess(ctx, httpData)
				if err != nil {
					httpData.Error = append(httpData.Error, err)
				}
			}

			if m.processor != nil {
				err := m.processor.Process(ctx, httpData)
				if err != nil {
					httpData.Error = append(httpData.Error, err)
				}
			}

			if m.errorHandler != nil && len(httpData.Error) > 0 {
				m.errorHandler.ErrorHandle(ctx, httpData.Error)
			}
		}()

		next(writerClone, request)
	})
}
