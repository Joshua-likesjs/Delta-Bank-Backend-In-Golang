package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    
    "delta-bank/internal/handler"
    "delta-bank/internal/repository"
    "delta-bank/internal/usecase"
    
    "github.com/gorilla/mux"
    "github.com/jackc/pgx/v5"
  
)

func main() {

    connStr := os.Getenv("DATABASE_URL")

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
    if port == "" { port = "8080" }

    fmt.Printf("\n🚀 Servidor em :%s\n", port)
    log.Fatal(http.ListenAndServe(":"+port, router))
}