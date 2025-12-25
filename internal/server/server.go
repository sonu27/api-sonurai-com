package server

import (
	"net/http"
	"time"

	rscors "github.com/rs/cors"
)

func New(port string, h http.Handler) http.Server {
	cors := rscors.New(rscors.Options{
		AllowedOrigins: []string{
			"https://sonurai.com",
			"https://*.vercel.app",
			"http://localhost:3000",
		},
		AllowCredentials: true,
		Debug:            false,
	})

	return http.Server{
		Addr:         ":" + port,
		Handler:      cors.Handler(h),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}
