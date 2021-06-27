package dipipe

// the API
// newMaster(a slice of Specs for the pipeline's stages) -> returns an instance of the pipeline
// Start(source spec, sink spec) -> return any fatal error ; Start is a blocking method
