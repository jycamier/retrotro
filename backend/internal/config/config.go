package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration
type Config struct {
	Port        int
	DatabaseURL string
	CORSOrigins []string
	DevMode     bool
	OIDC        OIDCConfig
	JWT         JWTConfig
	BusType string
	NatsURL string
}

// OIDCConfig holds OIDC provider configuration
type OIDCConfig struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	JIT          JITConfig
}

// JITConfig holds Just-In-Time provisioning configuration
type JITConfig struct {
	Enabled            bool
	GroupsClaim        string
	GroupsPrefix       string
	DefaultRole        string
	AdminGroups        []string
	FacilitatorGroups  []string
	SyncOnLogin        bool
	RemoveStaleMembers bool
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret          string
	AccessTokenTTL  int // minutes
	RefreshTokenTTL int // hours
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	port, _ := strconv.Atoi(getEnv("PORT", "8080"))
	accessTTL, _ := strconv.Atoi(getEnv("JWT_ACCESS_TOKEN_TTL", "15"))
	refreshTTL, _ := strconv.Atoi(getEnv("JWT_REFRESH_TOKEN_TTL", "168")) // 7 days

	return &Config{
		Port:        port,
		DatabaseURL: getEnv("DATABASE_URL", "postgres://retrotro:retrotro@localhost:5432/retrotro?sslmode=disable"),
		CORSOrigins: strings.Split(getEnv("CORS_ORIGINS", "http://localhost:3000"), ","),
		DevMode:     getEnv("DEV_MODE", "false") == "true",
		OIDC: OIDCConfig{
			IssuerURL:    getEnv("OIDC_ISSUER_URL", ""),
			ClientID:     getEnv("OIDC_CLIENT_ID", ""),
			ClientSecret: getEnv("OIDC_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("OIDC_REDIRECT_URL", "http://localhost:8080/auth/callback"),
			Scopes:       strings.Split(getEnv("OIDC_SCOPES", "openid,profile,email"), ","),
			JIT: JITConfig{
				Enabled:            getEnv("OIDC_JIT_ENABLED", "true") == "true",
				GroupsClaim:        getEnv("OIDC_JIT_GROUPS_CLAIM", "groups"),
				GroupsPrefix:       getEnv("OIDC_JIT_GROUPS_PREFIX", ""),
				DefaultRole:        getEnv("OIDC_JIT_DEFAULT_ROLE", "member"),
				AdminGroups:        strings.Split(getEnv("OIDC_JIT_ADMIN_GROUPS", ""), ","),
				FacilitatorGroups:  strings.Split(getEnv("OIDC_JIT_FACILITATOR_GROUPS", ""), ","),
				SyncOnLogin:        getEnv("OIDC_JIT_SYNC_ON_LOGIN", "true") == "true",
				RemoveStaleMembers: getEnv("OIDC_JIT_REMOVE_STALE_MEMBERS", "false") == "true",
			},
		},
		JWT: JWTConfig{
			Secret:          getEnv("JWT_SECRET", "change-me-in-production"),
			AccessTokenTTL:  accessTTL,
			RefreshTokenTTL: refreshTTL,
		},
		BusType: getEnv("BUS_TYPE", "gochannel"),
		NatsURL: getEnv("NATS_URL", ""),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
