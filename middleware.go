package hachibi

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"github.com/pkg/errors"
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

type KeyCtxMiddleware int

const (
	KeyErrorCtxMiddleware    = KeyCtxMiddleware(0)
	KeyHttpDataCtxMiddleware = KeyCtxMiddleware(1)
	KeyEventCtxMiddleware    = KeyCtxMiddleware(2)
	keyExtractData           = KeyCtxMiddleware(3)
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

type extractData bool

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

func getMiddlewareHttpData(w http.ResponseWriter, request *http.Request) (*HttpData, error) {
	httpData, ok := request.Context().Value(KeyHttpDataCtxMiddleware).(*HttpData)
	if !ok {
		return nil, errors.New("http data not exist")
	}

	extracted, ok := request.Context().Value(keyExtractData).(*extractData)
	if !ok {
		return nil, errors.New("no information about extracted data")
	}

	if !(*extracted) || httpData.StatusCode == 0 {
		writerClone, ok := w.(*Writer)
		if !ok {
			return nil, errors.New("cannot cast writer, you need to place preProcess after middleware")
		}

		httpData.URL = request.URL.String()
		httpData.Method = request.Method
		httpData.StatusCode = writerClone.statusCode

		httpData.Response = Response{Payload{
			Header: writerClone.w.Header().Clone(),
			Body:   writerClone.body.Bytes(),
		}}

		*extracted = true
	}

	return httpData, nil
}

func (m Middleware) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		timeStart := time.Now().Local()
		writerClone := newWriter(writer)
		httpData := HttpData{}
		extractD := extractData(false)

		err := httpData.extractRequest(request)
		if err != nil {
			httpData.Error = append(httpData.Error, err)
		}

		ctx = context.WithValue(ctx, keyExtractData, &extractD)
		ctx = context.WithValue(ctx, KeyHttpDataCtxMiddleware, &httpData)
		request = request.WithContext(ctx)

		defer func() {
			httpData, err := getMiddlewareHttpData(writerClone, request)
			if err != nil {
				return
			}

			ctx := request.Context()
			httpData.Duration = time.Since(timeStart).Nanoseconds()

			if err := getErrorInMiddlewareCtx(ctx); err != nil {
				httpData.Error = append(httpData.Error, err...)
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
	}
}

func (m Middleware) PreProcessMiddleware(preProcessor PreProcessor) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			next(writer, request)

			if preProcessor != nil {
				httpData, err := getMiddlewareHttpData(writer, request)
				if err != nil {
					return
				}

				err = preProcessor.PreProcess(request.Context(), httpData)
				if err != nil {
					httpData.Error = append(httpData.Error, err)
				}
			}
		})
	}
}

func (m Middleware) SetEventName(event string) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {

		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			next(writer, request)

			httpData, err := getMiddlewareHttpData(writer, request)
			if err != nil {
				return
			}

			httpData.Event = event
		})
	}
}
