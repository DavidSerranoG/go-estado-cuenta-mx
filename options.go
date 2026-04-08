package edocuenta

// Option configures a Processor.
type Option func(*Processor)

// WithExtractor overrides the default PDF text extractor.
func WithExtractor(extractor TextExtractor) Option {
	return func(p *Processor) {
		if extractor != nil {
			p.extractor = extractor
		}
	}
}

// WithRescueExtractor overrides the optional OCR rescue extractor used after a
// parser reports unsupported or insufficient text from the primary extraction.
//
// Rescue extraction is opt-in; the default processor does not enable OCR
// automatically.
func WithRescueExtractor(extractor TextExtractor) Option {
	return func(p *Processor) {
		if extractor != nil {
			p.rescueExtractor = extractor
		}
	}
}

// WithParser registers a single bank parser.
func WithParser(parser Parser) Option {
	return func(p *Processor) {
		if parser != nil {
			p.parsers.Add(parser)
		}
	}
}

// WithParsers registers multiple bank parsers.
func WithParsers(parsers ...Parser) Option {
	return func(p *Processor) {
		for _, parser := range parsers {
			if parser != nil {
				p.parsers.Add(parser)
			}
		}
	}
}
