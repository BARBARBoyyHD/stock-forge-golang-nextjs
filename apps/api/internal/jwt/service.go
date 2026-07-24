package jwt

type TokenService struct {
	Config TokenConfig
}

func New(config TokenConfig) *TokenService {
	return &TokenService{
		Config: config,
	}
}