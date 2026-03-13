package customrecexample

import maa "github.com/MaaXYZ/maa-framework-go/v4"

// Register registers the custom recognition example components in this package.
func Register() {
	maa.AgentServerRegisterCustomRecognition("CustomRecExampleMultiRecognition", &multiRecognition{})
}
