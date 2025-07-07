package main

import (
	"apiGateway/internal/config"
	"apiGateway/internal/http-server/middleware/mwlogger"
	"apiGateway/internal/lib/logger/handlers/slogpretty"
	"apiGateway/internal/lib/logger/sl"
	"context"
	"encoding/json"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
)

var votings = make(map[string]Voting)

type Voting struct {
	ID          string   `json:"voting_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	IsPrivate   bool     `json:"is_private"`
	MinVotes    int      `json:"min_votes"`
	EndDate     string   `json:"end_date"`
	Options     []string `json:"options"`
}

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("Starting voting service", slog.String("env", cfg.Env))
	log.Debug("Debug messages are enabled")

	_, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	wg.Add(1)

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(mwlogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/index.html")
	})
	router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	router.Post("/voting", CreateVoting)
	router.Get("/voting/{id}", GetVotingByID)
	router.Get("/voting", GetAllVotings)

	log.Info("starting server", slog.String("address", cfg.HTTPServer.Address))

	// Запускаем сервер
	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server", sl.Err(err))
	}

	// Graceful shutdown

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)

	sign := <-stop

	log.Info("application stopping", slog.String("signal", sign.String()))
	cancel()
	wg.Wait()

	log.Info("application stopped")
}

// setupLogger создает логгер с различными хендерами и уровнями логирования в зависимости от окружения
func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}

// setupPrettySlog создает логгер с удобным выводом данных для локала
func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	h := opts.NewPrettyHandler(os.Stdout)

	return slog.New(h)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func CreateVoting(w http.ResponseWriter, r *http.Request) {
	var newVoting Voting
	err := json.NewDecoder(r.Body).Decode(&newVoting)
	if err != nil {
		http.Error(w, "invalid request payload", http.StatusBadRequest)
		return
	}

	newVoting.ID = strconv.Itoa(len(votings) + 1) // simple ID

	votings[newVoting.ID] = newVoting

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"voting_id": newVoting.ID,
	})
}

func GetVotingByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	voting, ok := votings[id]
	if !ok {
		http.Error(w, "Voting not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(voting)
}

func GetAllVotings(w http.ResponseWriter, r *http.Request) {
	all := []Voting{}
	for _, v := range votings {
		all = append(all, v)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(all)
}
