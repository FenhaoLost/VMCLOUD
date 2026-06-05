package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"clicd/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

type ApiKey struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Key         string `json:"key,omitempty"`
	Prefix      string `json:"prefix"`
	IPWhitelist string `json:"ip_whitelist"`
	CreatedAt   string `json:"created_at"`
	LastUsed    string `json:"last_used"`
}

// HandleApiKeys handles GET (list) and POST (create) for API keys
func HandleApiKeys(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		listApiKeys(w, r)
	case http.MethodPost:
		createApiKey(w, r)
	default:
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
	}
}

// HandleApiKeyDelete handles DELETE for a specific API key
func HandleApiKeyDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Message: "Method not allowed"})
		return
	}
	keyID := strings.TrimPrefix(r.URL.Path, "/api/api-keys/")
	if keyID == "" {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Key ID required"})
		return
	}
	config.DeleteApiKey(keyID)
	jsonResponse(w, http.StatusOK, APIResponse{Success: true, Message: "API key deleted"})
}

func listApiKeys(w http.ResponseWriter, r *http.Request) {
	keys := make([]ApiKey, 0)
	for _, k := range config.AppConfig.ApiKeys {
		keys = append(keys, ApiKey{
			ID:          k.ID,
			Name:        k.Name,
			Prefix:      k.Prefix,
			IPWhitelist: k.IPWhitelist,
			CreatedAt:   k.CreatedAt,
			LastUsed:    k.LastUsed,
		})
	}
	jsonResponse(w, http.StatusOK, APIResponse{Success: true, Data: keys})
}

func createApiKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		IPWhitelist string `json:"ip_whitelist"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Message: "Name is required"})
		return
	}

	// Generate key: clicd_sk_ + 32 hex chars
	rawBytes := make([]byte, 16)
	rand.Read(rawBytes)
	rawKey := "clicd_sk_" + hex.EncodeToString(rawBytes)

	now := time.Now().Format("2006-01-02 15:04:05")
	key := config.ApiKeyConfig{
		ID:          generateShortID(),
		Name:        req.Name,
		KeyHash:     hashKey(rawKey),
		Prefix:      rawKey[:13] + "...",
		IPWhitelist: strings.TrimSpace(req.IPWhitelist),
		CreatedAt:   now,
	}
	config.AppConfig.ApiKeys = append(config.AppConfig.ApiKeys, key)
	config.SaveConfig()

	jsonResponse(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: "API key created. Save this key now - it won't be shown again.",
		Data: ApiKey{
			ID:          key.ID,
			Name:        key.Name,
			Key:         rawKey,
			Prefix:      key.Prefix,
			IPWhitelist: key.IPWhitelist,
			CreatedAt:   key.CreatedAt,
		},
	})
}

func generateShortID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// hashKey creates a simple hash for storage (not reversible)
func hashKey(key string) string {
	b := make([]byte, 32)
	for i := range key {
		b[i%32] ^= key[i]
	}
	return hex.EncodeToString(b)
}

// validateApiKey checks if the given key is valid and IP is allowed
func validateApiKey(rawKey, clientIP string) bool {
	hashed := hashKey(rawKey)
	for _, k := range config.AppConfig.ApiKeys {
		if k.KeyHash == hashed {
			if k.IPWhitelist == "" {
				return true
			}
			return isIPAllowed(clientIP, k.IPWhitelist)
		}
	}
	return false
}

// isIPAllowed checks if clientIP matches any entry in the whitelist
func isIPAllowed(clientIP, whitelist string) bool {
	clientIP = strings.TrimSpace(clientIP)
	// Strip port if present
	if idx := strings.LastIndex(clientIP, ":"); idx > strings.LastIndex(clientIP, "]") {
		clientIP = clientIP[:idx]
	}
	for _, entry := range strings.Split(whitelist, "\n") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			// CIDR match
			if ipInCIDR(clientIP, entry) {
				return true
			}
		} else if entry == clientIP {
			return true
		}
	}
	return false
}

func ipInCIDR(ipStr, cidr string) bool {
	parts := strings.Split(cidr, "/")
	if len(parts) != 2 {
		return false
	}
	// Simple prefix match for IPv4
	ip := netParseIP(ipStr)
	cidrIP := netParseIP(parts[0])
	if ip == nil || cidrIP == nil {
		return false
	}
	bits, err := strconv.Atoi(parts[1])
	if err != nil || bits < 0 || bits > 32 {
		return false
	}
	mask := uint32(0xFFFFFFFF) << (32 - bits)
	ipVal := ip4ToUint32(ip)
	cidrVal := ip4ToUint32(cidrIP)
	return (ipVal & mask) == (cidrVal & mask)
}

func netParseIP(s string) net.IP {
	s = strings.TrimSpace(s)
	if idx := strings.LastIndex(s, ":"); idx > strings.LastIndex(s, "]") {
		s = s[:idx]
	}
	return net.ParseIP(s)
}

func ip4ToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

// updateApiKeyLastUsed marks the key as recently used
func updateApiKeyLastUsed(rawKey string) {
	hashed := hashKey(rawKey)
	now := time.Now().Format("2006-01-02 15:04:05")
	for i := range config.AppConfig.ApiKeys {
		if config.AppConfig.ApiKeys[i].KeyHash == hashed {
			config.AppConfig.ApiKeys[i].LastUsed = now
			config.SaveConfig()
			return
		}
	}
}

// ApiKeyMiddleware authenticates requests via X-API-Key header or ?api_key query param
func ApiKeyMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			// Check query param
			apiKey = r.URL.Query().Get("api_key")
		}
		if apiKey == "" {
			// Check Bearer token (some clients use this)
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer clicd_sk_") {
				apiKey = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		// Get client IP
		clientIP := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			clientIP = strings.Split(forwarded, ",")[0]
		}
		if apiKey == "" || !validateApiKey(apiKey, clientIP) {
			jsonResponse(w, http.StatusUnauthorized, APIResponse{Success: false, Message: "Invalid API key or IP not in whitelist"})
			return
		}

		// Generate a short-lived JWT so downstream admin middleware passes
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": config.AppConfig.AdminUser,
			"api_key":  true,
			"exp":      time.Now().Add(5 * time.Minute).Unix(),
			"iat":      time.Now().Unix(),
		})
		tokenString, _ := token.SignedString([]byte(config.AppConfig.JWTSecret))

		// Set cookie for subsequent requests
		http.SetCookie(w, &http.Cookie{
			Name:     "clicd_token",
			Value:    tokenString,
			Path:     "/",
			HttpOnly: false,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   300,
		})

		updateApiKeyLastUsed(apiKey)
		next(w, r)
	}
}
