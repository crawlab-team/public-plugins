package entity

type ScrapyCfg struct {
	Settings map[string]string `json:"settings"`
	Deploy   map[string]string `json:"deploy"`
}

func NewScrapyCfg() *ScrapyCfg {
	return &ScrapyCfg{
		Settings: make(map[string]string),
		Deploy:   make(map[string]string),
	}
}
