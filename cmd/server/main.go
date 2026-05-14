package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"delta-bank/internal/handler"
	"delta-bank/internal/repository"
	"delta-bank/internal/usecase"

  //  "github.com/joho/godotenv"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
)

func main() {
	// FIX 2: valida DATABASE_URL antes de tentar conectar
  //  _ = godotenv.Load()
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL não configurada")
	}

	db, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Erro banco: %v\n", err)
	}
	defer db.Close(context.Background())

	fmt.Println(` 🌐 API REST - CLEAN ARCHITECTURE v9.0 `)

	repo := repository.NovoPostgresRepo(db)
	uc := usecase.NovoServico(repo)
	hdl := handler.NovoAPIHandler(uc)

	router := mux.NewRouter()

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/contas", hdl.CriarConta).Methods("POST", "OPTIONS")
	api.HandleFunc("/login", hdl.Login).Methods("POST", "OPTIONS")
	api.HandleFunc("/saldo/{cpf}", hdl.Saldo).Methods("GET")
	api.HandleFunc("/extrato/{cpf}", hdl.Extrato).Methods("GET")
	api.HandleFunc("/pix", hdl.Pix).Methods("POST", "OPTIONS")
	api.HandleFunc("/depositar", hdl.Depositar).Methods("POST", "OPTIONS")
	api.HandleFunc("/sacar", hdl.Sacar).Methods("POST", "OPTIONS")
	api.HandleFunc("/chaves-pix/{cpf}", hdl.ListarChaves).Methods("GET")
	api.HandleFunc("/chaves-pix", hdl.AdicionarChave).Methods("POST", "OPTIONS")
	api.HandleFunc("/chaves-pix", hdl.RemoverChave).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/senha", hdl.MudarSenha).Methods("PUT", "OPTIONS")
	api.HandleFunc("/dados", hdl.EditarDados).Methods("PUT", "OPTIONS")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// FIX 3: graceful shutdown — conexões ativas têm até 10s para terminar
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		fmt.Printf("\n🚀 Servidor em :%s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Erro servidor: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n⏳ Encerrando servidor...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Erro ao encerrar: %v\n", err)
	}
	fmt.Println("✅ Servidor encerrado.")
}