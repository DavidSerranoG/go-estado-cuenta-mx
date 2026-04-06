package statementpdf

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
