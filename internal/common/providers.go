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
			Name:       "GitHub Models",
			ProviderID: "github-models",
			BaseURL:    "https://models.inference.ai.azure.com",
			Models: []Model{
				Model{
					Name:    "GPT-4.1 Mini",
					ModelID: "gpt-4-1-mini",
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
			BaseURL:    "https://api.moonshot.cn/v1",
			Models: []Model{
				Model{
					Name:    "Kimi K2",
					ModelID: "kimi-k2-turbo-preview",
				},
			},
		},
		&Provider{
			Name:       "OpenAI",
			ProviderID: "openai",
			BaseURL:    "https://api.openai.com/v1",
			Models: []Model{
				Model{
					Name:    "GPT-5.6 Terra",
					ModelID: "gpt-5.6-terra",
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
