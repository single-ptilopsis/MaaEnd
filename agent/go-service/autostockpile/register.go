package autostockpile

import maa "github.com/MaaXYZ/maa-framework-go/v4"

// Register registers all custom action and recognition components for autostockpile package
func Register() {
	maa.AgentServerRegisterCustomAction("AutoStockpile.SelectItem", &SelectItemAction{})
	maa.AgentServerRegisterCustomRecognition("AutoStockpile.Recognition", &ItemValueChangeRecognition{})
}
