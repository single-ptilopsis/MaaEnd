package stockpile

import "github.com/MaaXYZ/maa-framework-go/v4"

// Register registers all custom action and recognition components
// for the stockpile package with the MaaFramework agent server.
func Register() {
	maa.AgentServerRegisterCustomRecognition("StockpileCheckQuotaRecognition", &StockpileCheckQuotaRecognition{})
	maa.AgentServerRegisterCustomAction("StockpileInitAction", &StockpileInitAction{})
	maa.AgentServerRegisterCustomAction("StockpileCheckQuotaAction", &StockpileCheckQuotaAction{})
	maa.AgentServerRegisterCustomAction("StockpileScanAction", &StockpileScanAction{})
	maa.AgentServerRegisterCustomAction("StockpileScanPriceAction", &StockpileScanPriceAction{})
	maa.AgentServerRegisterCustomAction("StockpileScanSkipEmptyAction", &StockpileScanSkipEmptyAction{})
	maa.AgentServerRegisterCustomAction("StockpileScanNextAction", &StockpileScanNextAction{})
	maa.AgentServerRegisterCustomAction("StockpileDecideAction", &StockpileDecideAction{})
	maa.AgentServerRegisterCustomAction("StockpileFinishAction", &StockpileFinishAction{})
}
