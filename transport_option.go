package hachibi

type TransportOpt func(*Transport)

func TransportWithProcessor(processor Processor) TransportOpt {
	return func(transport *Transport) {
		transport.processor = processor
	}
}

func TransportWithEventName(name string) TransportOpt {
	return func(transport *Transport) {
		transport.Event = name
	}
}

func TransportWithErrorHandler(handler ErrorHandler) TransportOpt {
	return func(transport *Transport) {
		transport.errorHandler = handler
	}
}

func TransportWithPreProcessor(p PreProcessor) TransportOpt {
	return func(transport *Transport) {
		transport.preProcessor = p
	}
}

func TransportWithPostProcessor(p PostProcessor) TransportOpt {
	return func(transport *Transport) {
		transport.postProcessor = p
	}
}
