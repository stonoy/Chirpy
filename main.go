package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/lpernett/godotenv"
)

type apiConfig struct {
	fileserverHits int
	DB             *DB
	jwtkey         string
	apiKey         string
}

func main() {
	// Load env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	apiKeyHook := os.Getenv("API_KEY")

	// creating new database if database not exists
	myDb, err := NewDb("./database.json")
	if err != nil {
		fmt.Printf("Database not configured -->> %v\n", err)
	}

	apiCfg := &apiConfig{
		fileserverHits: 0,
		DB:             myDb,
		jwtkey:         jwtSecret,
		apiKey:         apiKeyHook,
	}

	shareFolderPath := "."
	port := "8080"

	// mux := http.NewServeMux()

	router := chi.NewRouter()

	rFunc := apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(shareFolderPath))))

	router.Handle("/app", rFunc)
	router.Handle("/app/*", rFunc)

	// mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
	// 	line := fmt.Sprintf("Hits: %v\n", apiCfg.fileserverHits)
	// 	w.Write([]byte(line))
	// })
	apiRouter := chi.NewRouter()
	apiRouter.Get("/healthz", handelReadyNess())
	apiRouter.Get("/reset", apiCfg.resetHits())

	// All chirps api
	apiRouter.Post("/chirps", postReqChirp(apiCfg))
	apiRouter.Get("/chirps", getReqChirp(apiCfg.DB))
	apiRouter.Get("/chirps/{chirpsId}", getReqOneChirp(apiCfg.DB))
	apiRouter.Delete("/chirps/{chirpsId}", deleteChirp(apiCfg))

	// All user api
	apiRouter.Post("/users", postReqUser(apiCfg))
	apiRouter.Put("/users", putUsers(apiCfg))

	// Login route
	apiRouter.Post("/login", loginUser(apiCfg))

	// Refresh Tokens
	apiRouter.Post("/revoke", revokeTokens(apiCfg))
	apiRouter.Post("/refresh", refreshTokens(apiCfg))

	// WebHooks
	apiRouter.Post("/polka/webhooks", handelWebHooks(apiCfg))

	router.Mount("/api", apiRouter)

	adminRouter := chi.NewRouter()
	adminRouter.Get("/metrics", apiCfg.sendHits())

	router.Mount("/admin", adminRouter)

	corsMux := middlewareCors(router)

	s := &http.Server{
		Addr:    ":" + port,
		Handler: corsMux,
	}

	s.ListenAndServe()
}
