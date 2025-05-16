package config

type AppConfig struct {
	ShortPathLength           int
	UnguessablePathLength     int
	DefaultAndroidPackageName *string
	DefaultIosStoreId         *string
	URLScheme                 string
	AllowedDomains            []string
}

func NewAppConfig() *AppConfig {
	return &AppConfig{
		ShortPathLength:           getEnvAsInt("SHORT_PATH_LENGTH", 6),
		UnguessablePathLength:     getEnvAsInt("UNGUESSABLE_PATH_LENGTH", 10),
		DefaultAndroidPackageName: getEnvAsOptionalString("DEFAULT_ANDROID_PACKAGE_NAME"),
		DefaultIosStoreId:         getEnvAsOptionalString("DEFAULT_IOS_STORE_ID"),
		URLScheme:                 getEnv("URL_SCHEME", "https"),
		AllowedDomains:            getEnvAsSlice("ALLOWED_DOMAINS", []string{}),
	}
}
