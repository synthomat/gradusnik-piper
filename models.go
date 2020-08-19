package piper

type Aqara struct {
	Temperature float32 `json:"temperature"`
	Humidity    float32 `json:"humidity"`
	Pressure    float32 `json:"pressure"`
	Battery     float32 `json:"battery"`
}
