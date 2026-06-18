package providers

// KeyPoolInterface interface để KeyPool có thể được inject từ core
type KeyPoolInterface interface {
	GetNextKey() (string, int)
	ReleaseKey(key string)
}

// GeminiProviderWithPool extends GeminiProvider để support key pool
type GeminiProviderWithPool struct {
	*GeminiProvider
	keyPool KeyPoolInterface
}