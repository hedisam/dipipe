// this is for testing the plugin builder
// the plugin code needs to have at least one 'main' package

// this is an example of a pipeline's processor which will be compiled as a plugin. the Dockerfile, go mod files &
// vendor dir along with the .so files are generated by the library (to be specific, by the plugin builder)

// an example of user's code for a pipeline's processor

package main

// the actual Process function defined by the user must match the models.Processor, this one is just for showing an
// example of a plugin's source code directory
func Process(msg string) (string, error) {
	return "processed:"+msg, nil
}
