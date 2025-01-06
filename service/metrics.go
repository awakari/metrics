package service

type NumberHistory struct {
	Current float64    `json:"current"`
	Past    NumberPast `json:"past"`
}

type NumberPast struct {
	Hour  float64 `json:"hour"`
	Day   float64 `json:"day"`
	Month float64 `json:"month"`
}

type Attributes struct {
	TypesByKey map[string][]string `json:"typesByKey"`
}

type Duration struct {
	Quantile05  float64 `json:"q0_5"`
	Quantile075 float64 `json:"q0_75"`
	Quantile095 float64 `json:"q0_95"`
	Quantile099 float64 `json:"q0_99"`
}

type ReadStatus struct {
	ReadRate        float64            `json:"readRate"`
	SourcesMostRead map[string]float64 `json:"sourcesMostRead"`
}
