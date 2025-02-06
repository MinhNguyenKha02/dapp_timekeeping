package utils

var Config struct {
	JWTSecret string
}

func init() {
	// Set a default secret for development
	Config.JWTSecret = "nB9sMCWS7anqMGuWEPefT+tt68T+tAdiCoBkfx2H9Oc="
}
