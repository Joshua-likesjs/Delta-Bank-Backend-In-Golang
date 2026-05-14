package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"delta-bank/internal/handler"
	"delta-bank/internal/middleware"
	"delta-bank/internal/repository"
	"delta-bank/internal/usecase"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Logger estruturado — saída JSON em produção é parseable por qualquer
	// aggregator (Datadog, Loki, CloudWatch, etc.).
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(log)

	// Carrega .env se existir (silencioso em produção onde as vars já estão definidas).
	_ = godotenv.Load()

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Error("DATABASE_URL não configurada")
		os.Exit(1)
	}

	// pgxpool em vez de pgx.Conn — gerencia um pool de conexões para
	// suportar múltiplas requisições concorrentes em produção.
	poolCfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Error("configuração do pool inválida", "error", err)
		os.Exit(1)
	}
	poolCfg.MaxConns = 20
	poolCfg.MinConns = 2
	poolCfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		log.Error("erro ao conectar ao banco", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Error("banco inacessível", "error", err)
		os.Exit(1)
	}
	log.Info("conectado ao banco de dados", "max_conns", poolCfg.MaxConns)

	// Wiring das camadas (dependency injection manual).
	repo := repository.NovoPostgresRepo(pool)
	svc := usecase.NovoServico(repo)
	hdl := handler.NovoAPIHandler(svc)

	router := buildRouter(hdl, log)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Inicia o servidor numa goroutine e aguarda sinal de encerramento.
	go func() {
		fmt.Printf("\n🌐 Delta Bank API — porta :%s\n\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("erro fatal no servidor", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Info("sinal recebido, encerrando servidor", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("erro ao encerrar servidor", "error", err)
		os.Exit(1)
	}

	log.Info("servidor encerrado")
}

func buildRouter(hdl *handler.APIHandler, log *slog.Logger) *mux.Router {
	r := mux.NewRouter()

	// Middleware stack aplicado na ordem correta:
	// RequestID → Logger → Recovery → Timeout → CORS → MaxBodyBytes → rotas
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(log))
	r.Use(middleware.Recovery(log))
	r.Use(middleware.Timeout(25 * time.Second))
	r.Use(middleware.CORS())
	r.Use(middleware.MaxBodyBytes(1 << 20)) // 1 MB

	// Health check — sem prefixo /api para ser chamado por load balancers.
	r.HandleFunc("/health", hdl.Health).Methods(http.MethodGet)

	api := r.PathPrefix("/api").Subrouter()

	// Contas
	api.HandleFunc("/contas", hdl.CriarConta).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/login", hdl.Login).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/saldo/{cpf}", hdl.Saldo).Methods(http.MethodGet)
	api.HandleFunc("/extrato/{cpf}", hdl.Extrato).Methods(http.MethodGet)
	api.HandleFunc("/dados", hdl.EditarDados).Methods(http.MethodPut, http.MethodOptions)
	api.HandleFunc("/senha", hdl.MudarSenha).Methods(http.MethodPut, http.MethodOptions)

	// Operações financeiras
	api.HandleFunc("/pix", hdl.Pix).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/depositar", hdl.Depositar).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/sacar", hdl.Sacar).Methods(http.MethodPost, http.MethodOptions)

	// Chaves PIX
	api.HandleFunc("/chaves-pix/{cpf}", hdl.ListarChaves).Methods(http.MethodGet)
	api.HandleFunc("/chaves-pix", hdl.AdicionarChave).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/chaves-pix", hdl.RemoverChave).Methods(http.MethodDelete, http.MethodOptions)

	return r
}