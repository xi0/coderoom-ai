package common

type Provider struct {
	Name       string
	ProviderID string
	BaseURL    string
	Models     []Model
}

type Model struct {
	Name    string
	ModelID string
}

type ProviderModel struct {
	ProviderName string
	ModelName    string
	BaseURL      string
}

var (
	Providers = []*Provider{
		&Provider{
			Name:       "DeepSeek",
			ProviderID: "deepseek",
			BaseURL:    "https://api.deepseek.com",
			Models: []Model{
				Model{
					Name:    "DeepSeek Flash",
					ModelID: "deepseek-flash",
				},
				Model{
					Name:    "DeepSeek V4 Pro",
					ModelID: "deepseek-v4-pro",
				},
			},
		},
		&Provider{
			Name:       "Depaza",
			ProviderID: "depaza",
			BaseURL:    "https://depaza.com/v1",
			Models: []Model{
				Model{
					Name:    "Max",
					ModelID: "max",
				},
				Model{
					Name:    "Advanced",
					ModelID: "advanced",
				},
			},
		},
		&Provider{
			Name:       "GDPRchat",
			ProviderID: "gdprchat",
			BaseURL:    "https://gdprchat.eu/api/v1",
			Models: []Model{
				Model{
					Name:    "Code",
					ModelID: "code",
				},
			},
		},
		&Provider{
			Name:       "Kimi (Moonshot AI)",
			ProviderID: "kimi",
			BaseURL:    "https://api.moonshot.ai/v1",
			Models: []Model{
				Model{
					Name:    "Kimi K3",
					ModelID: "kimi-k3",
				},
			},
		},
		&Provider{
			Name:       "OpenAI",
			ProviderID: "openai",
			BaseURL:    "https://api.openai.com/v1",
			Models: []Model{
				Model{
					Name:    "GPT-6 Luna",
					ModelID: "gpt-6-luna",
				},
			},
		},
	}
	ProvidersMap map[string]map[string]*ProviderModel
)

func init() {
	ProvidersMap = make(map[string]map[string]*ProviderModel)

	for _, p := range Providers {
		for _, m := range p.Models {
			if _, ok := ProvidersMap[p.ProviderID]; !ok {
				ProvidersMap[p.ProviderID] = make(map[string]*ProviderModel)
			}

			ProvidersMap[p.ProviderID][m.ModelID] = &ProviderModel{
				ProviderName: p.Name,
				ModelName:    m.Name,
				BaseURL:      p.BaseURL,
			}
		}
	}
}
