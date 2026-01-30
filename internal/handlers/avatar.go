package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"image"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"octa/internal/config"
	"octa/internal/database"

	"octa/pkg/generator"
	"octa/pkg/generator/styles"
	"octa/pkg/utils"
)

func buildCacheKey(prefix string, key string, query url.Values) (string, bool) {
	// Skip caching for custom colors to prevent cache pollution (DoS protection)
	if query.Get("bg") != "" || query.Get("color") != "" {
		return fmtKey(prefix, key, query), false
	}

	return fmtKey(prefix, key, query), true
}

func fmtKey(prefix, key string, query url.Values) string {
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys) // Ensure consistent key order

	var sb strings.Builder
	sb.WriteString(prefix)
	sb.WriteString(":")
	sb.WriteString(key)
	sb.WriteString("?")
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(query.Get(k))
		sb.WriteString("&")
	}
	return sb.String()
}

// serveWithETag handles HTTP caching headers (ETag, Cache-Control).
// Returns 304 Not Modified if client's cache is valid.
func serveWithETag(w http.ResponseWriter, r *http.Request, data []byte, mimeType string) {
	hash := sha256.Sum256(data)
	etag := hex.EncodeToString(hash[:])

	if mimeType == "" {
		mimeType = "image/png"
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("ETag", `"`+etag+`"`)

	if match := r.Header.Get("If-None-Match"); match != "" {
		if strings.Contains(match, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	w.Write(data)
}

// ServeDirectAvatar generates an avatar deterministically from the seed.
// Path: /avatar/:seed
func ServeDirectAvatar(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/avatar/")
	if key == "" {
		utils.WriteError(w, http.StatusBadRequest, utils.ErrRequestMissingKey, "Avatar seed key is missing.")
		return
	}

	uniqueKey, shouldCache := buildCacheKey("gen", key, r.URL.Query())

	// Execute generation within SingleFlight to optimize concurrent requests
	data, err, _ := requestGroup.Do(uniqueKey, func() (interface{}, error) {
		if shouldCache {
			if cached, ok := globalCache.Get(uniqueKey); ok {
				return cached, nil
			}
		}

		genData, _, err := styles.GenerateImageBytes(key, r.URL.Query())

		if err != nil {
			return nil, err
		}

		if shouldCache {
			globalCache.Set(uniqueKey, genData)
		}
		return genData, nil
	})

	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, utils.ErrImageGenerationFailed, "Failed to generate avatar image.")
		return
	}

	mime := "image/png"
	if r.URL.Query().Get("format") == "svg" || r.URL.Query().Get("type") == "svg" {
		mime = "image/svg+xml"
	}

	serveWithETag(w, r, data.([]byte), mime)
}

// ServeUserAvatar serves avatars from DB if available, otherwise falls back to generator.
// Path: /u/:key
func ServeUserAvatar(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/u/")
	if key == "" {
		utils.WriteError(w, http.StatusBadRequest, utils.ErrRequestMissingKey, "User identifier is missing.")
		return
	}

	var targetImageID string

	mapCacheKey := "map:" + key

	if cachedIDBytes, ok := globalCache.Get(mapCacheKey); ok {
		targetImageID = string(cachedIDBytes)
	} else {
		var mapping database.KeyMapping

		if err := database.DB.Select("image_id").First(&mapping, "key = ?", key).Error; err != nil {

			serveGeneratorFallback(w, r, key)
			return
		}
		targetImageID = mapping.ImageID

		globalCache.Set(mapCacheKey, []byte(targetImageID))
	}

	imgCacheKey := "img:" + targetImageID

	// DB Fetch
	sfDBGroupKey := "fetch_img:" + targetImageID
	data, dbError, _ := requestGroup.Do(sfDBGroupKey, func() (interface{}, error) {
		// Double-check cache inside lock
		if cached, ok := globalCache.Get(imgCacheKey); ok {
			return cached, nil
		}

		var mapping database.KeyMapping
		if err := database.DB.First(&mapping, "key = ?", key).Error; err != nil {
			return nil, err // Not found
		}

		var imgModel database.Image
		if err := database.DB.Select("data").First(&imgModel, "id = ?", mapping.ImageID).Error; err != nil {
			return nil, err
		}

		globalCache.Set(imgCacheKey, imgModel.Data)
		return imgModel.Data, nil
	})

	if dbError != nil {
		serveGeneratorFallback(w, r, key)
	}

	serveWithETag(w, r, data.([]byte), "image/png")

}

func serveGeneratorFallback(w http.ResponseWriter, r *http.Request, key string) {
	// Generator Fallback (If not in DB)
	uniqueKey, shouldCache := buildCacheKey("gen", key, r.URL.Query())

	genRes, genErr, _ := requestGroup.Do(uniqueKey, func() (interface{}, error) {
		if shouldCache {
			if cached, ok := globalCache.Get(uniqueKey); ok {
				return cached, nil
			}
		}

		genData, _, err := styles.GenerateImageBytes(uniqueKey, r.URL.Query())

		if err != nil {
			return nil, err
		}
		if shouldCache {
			globalCache.Set(uniqueKey, genData)
		}
		return genData, nil
	})

	if genErr != nil {
		utils.WriteError(w, http.StatusInternalServerError, utils.ErrImageGenerationFailed, "Unable to generate fallback avatar.")
		return
	}

	mimeType := "image/png"
	if r.URL.Query().Get("format") == "svg" {
		mimeType = "image/svg+xml"
	}

	serveWithETag(w, r, genRes.([]byte), mimeType)
}

// GITHUB AVATAR (/avatar/github/:username)
// By reducing the size of GitHub images by 75%, they will be delivered faster and your website's loading speed will increase significantly. Additionally, OCTA's custom generator creates beautiful avatars instead of GitHub's old, silly fallback user profiles.
func GithubAvatarHandler(w http.ResponseWriter, r *http.Request) {
	// Username Parse
	path := strings.TrimPrefix(r.URL.Path, "/avatar/github/")
	username := strings.Split(path, "/")[0]
	if username == "" {
		utils.WriteError(w, http.StatusBadRequest, utils.ErrRequestInvalid, "Username is required.")
		return
	}

	uniqueKey := "gh:" + username

	avatarSize := config.AppConfig.Image.DefaultSize
	if avatarSize == 0 {
		avatarSize = styles.DefaultAvatarSize
	}

	data, err, _ := requestGroup.Do(uniqueKey, func() (interface{}, error) {
	
		if cached, ok := globalCache.Get(uniqueKey); ok {
			return cached, nil
		}

		// genParams := url.Values{}
		// genParams.Set("size", fmt.Sprintf("%d", styles.DefaultAvatarSize)) // "360"

		// GitHub Metadata Fetch
		ghUser, err := generator.FetchGitHubName(username)

		fallbackName := username
		if err == nil && ghUser.Name != "" {
			fallbackName = ghUser.Name
		}

		if err != nil || ghUser.AvatarURL == "" {
			genData, _, genErr := styles.GenerateImageBytes(fallbackName, nil)
			if genErr == nil {
				globalCache.Set(uniqueKey, genData)
			}
			return genData, genErr
		}

		// Download Image
		imgResp, err := http.Get(ghUser.AvatarURL)
		if err != nil || imgResp.StatusCode != 200 {
			genData, _, genErr := styles.GenerateImageBytes(fallbackName, nil)

			if genErr == nil {
				globalCache.Set(uniqueKey, genData)
			}
			return genData, genErr
		}
		defer imgResp.Body.Close()

	
		img, _, err := image.Decode(imgResp.Body)
		if err != nil {
			return nil, err
		}

		procOpts := utils.ProcessOptions{
			Mode:    "fit",
			Size:    avatarSize,
			Quality: 80,
		}

		// Process
		processedBuf, _, _, err := utils.ProcessImage(img, procOpts)
		if err != nil {
			return nil, err
		}

		finalBytes := processedBuf.Bytes()

		globalCache.Set(uniqueKey, finalBytes)

		return finalBytes, nil
	})

	if err != nil {
		utils.WriteError(w, http.StatusBadGateway, utils.ErrUpstreamFailed, "Failed to process avatar.")
		return
	}

	serveWithETag(w, r, data.([]byte), "image/jpeg")
}
