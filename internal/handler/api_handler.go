package handler

import (
    "encoding/json"
    "net/http"

    "delta-bank/internal/usecase"
	"github.com/gorilla/mux"
)

type APIHandler struct {
    uc *usecase.Service
}

func NovoAPIHandler(uc *usecase.Service) *APIHandler {
    return &APIHandler{uc: uc}
}

func (h *APIHandler) SetHeaders(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
    if r.Method == "OPTIONS" { w.WriteHeader(http.StatusOK) }
}

// POST /api/contas
func (h *APIHandler) CriarConta(w http.ResponseWriter, r *http.Request) {
    h.SetHeaders(w, r)
    var req ReqCriarConta
    json.NewDecoder(r.Body).Decode(&req)

    conta, err := h.uc.CriarConta(r.Context(), usecase.CriarContaInput{
        Nome: req.Nome, CPF: req.CPF, Senha: req.Senha, Saldo: req.Saldo,
    })
    if err != nil {
        json.NewEncoder(w).Encode(RespostaAPI{Sucesso: false, Mensagem: err.Error()})
        return
    }
    json.NewEncoder(w).Encode(RespostaAPI{Sucesso: true, Mensagem: "Conta criada!", Dados: conta})
}

// POST /api/login
func (h *APIHandler) Login(w http.ResponseWriter, r *http.Request) {
    h.SetHeaders(w, r)
    var req ReqLogin
    json.NewDecoder(r.Body).Decode(&req)

    conta, err := h.uc.Login(r.Context(), usecase.LoginInput{CPF: req.CPF, Senha: req.Senha})
    if err != nil {
        json.NewEncoder(w).Encode(RespostaAPI{Sucesso: false, Mensagem: err.Error()})
        return
    }
    json.NewEncoder(w).Encode(RespostaAPI{Sucesso: true, Mensagem: "Login OK!", Dados: conta})
}

// GET /api/saldo/{cpf}
func (h *APIHandler) Saldo(w http.ResponseWriter, r *http.Request) {
    h.SetHeaders(w, r)
    params := mux.Vars(r)
    output, err := h.uc.ConsultarSaldo(r.Context(), params["cpf"])
    if err != nil {
        json.NewEncoder(w).Encode(RespostaAPI{Sucesso: false, Mensagem: err.Error()})
        return
    }
    json.NewEncoder(w).Encode(RespostaAPI{Sucesso: true, Dados: output})
}

// GET /api/extrato/{cpf}
func (h *APIHandler) Extrato(w http.ResponseWriter, r *http.Request) {
    h.SetHeaders(w, r)
    params := mux.Vars(r)
    transacoes, saldo, err := h.uc.Extrato(r.Context(), params["cpf"])
    if err != nil {
        json.NewEncoder(w).Encode(RespostaAPI{Sucesso: false, Mensagem: err.Error()})
        return
    }
    json.NewEncoder(w).Encode(RespostaAPI{Sucesso: true, Dados: map[string]interface{}{
        "transacoes": transacoes, "saldo_atual": saldo,
    }})
}

// POST /api/pix
func (h *APIHandler) Pix(w http.ResponseWriter, r *http.Request) {
    h.SetHeaders(w, r)
    var req ReqPix
    json.NewDecoder(r.Body).Decode(&req)

    tx, err := h.uc.FazerPix(r.Context(), usecase.PixInput{
        CPFOrigem: req.CPFOrigem, CPFDestino: req.CPFDestino, Valor: req.Valor,
    })
    if err != nil {
        json.NewEncoder(w).Encode(RespostaAPI{Sucesso: false, Mensagem: err.Error()})
        return
    }
    json.NewEncoder(w).Encode(RespostaAPI{Sucesso: true, Mensagem: "PIX realizado!", Dados: tx})
}

// POST /api/depositar
func (h *APIHandler) Depositar(w http.ResponseWriter, r *http.Request) {
    h.SetHeaders(w, r)
    var req ReqDeposito
    json.NewDecoder(r.Body).Decode(&req)

    novoSaldo, err := h.uc.Depositar(r.Context(), usecase.DepositoInput{CPF: req.CPF, Valor: req.Valor})
    if err != nil {
        json.NewEncoder(w).Encode(RespostaAPI{Sucesso: false, Mensagem: err.Error()})
        return
    }
    json.NewEncoder(w).Encode(RespostaAPI{Sucesso: true, Mensagem: "Depósito OK!", Dados: map[string]int{"novo_saldo": novoSaldo}})
}

// POST /api/sacar
func (h *APIHandler) Sacar(w http.ResponseWriter, r *http.Request) {
    h.SetHeaders(w, r)
    var req ReqSaque
    json.NewDecoder(r.Body).Decode(&req)

    novoSaldo, err := h.uc.Sacar(r.Context(), usecase.SaqueInput{CPF: req.CPF, Valor: req.Valor})
    if err != nil {
        json.NewEncoder(w).Encode(RespostaAPI{Sucesso: false, Mensagem: err.Error()})
        return
    }
    json.NewEncoder(w).Encode(RespostaAPI{Sucesso: true, Mensagem: "Saque OK!", Dados: map[string]int{"novo_saldo": novoSaldo}})
}

// GET /api/chaves-pix/{cpf}
func (h *APIHandler) ListarChaves(w http.ResponseWriter, r *http.Request) {
    h.SetHeaders(w, r)
    params := mux.Vars(r)
    chaves, err := h.uc.ListarChaves(r.Context(), params["cpf"])
    if err != nil {
        json.NewEncoder(w).Encode(RespostaAPI{Sucesso: false, Mensagem: err.Error()})
        return
    }
    json.NewEncoder(w).Encode(RespostaAPI{Sucesso: true, Dados: chaves})
}

// POST /api/chaves-pix
func (h *APIHandler) AdicionarChave(w http.ResponseWriter, r *http.Request) {
    h.SetHeaders(w, r)
    var req ReqAdicionarChave
    json.NewDecoder(r.Body).Decode(&req)

    chave, err := h.uc.AdicionarChave(r.Context(), usecase.AdicionarChaveInput{CPF: req.CPF, Tipo: req.Tipo, Valor: req.Valor})
    if err != nil {
        json.NewEncoder(w).Encode(RespostaAPI{Sucesso: false, Mensagem: err.Error()})
        return
    }
    json.NewEncoder(w).Encode(RespostaAPI{Sucesso: true, Mensagem: "Chave adicionada!", Dados: chave})
}

// DELETE /api/chaves-pix
func (h *APIHandler) RemoverChave(w http.ResponseWriter, r *http.Request) {
    h.SetHeaders(w, r)
    var req ReqRemoverChave
    json.NewDecoder(r.Body).Decode(&req)

    err := h.uc.RemoverChave(r.Context(), usecase.RemoverChaveInput{CPF: req.CPF, Tipo: req.Tipo, Valor: req.Valor})
    if err != nil {
        json.NewEncoder(w).Encode(RespostaAPI{Sucesso: false, Mensagem: err.Error()})
        return
    }
    json.NewEncoder(w).Encode(RespostaAPI{Sucesso: true, Mensagem: "Chave removida!"})
}

// PUT /api/senha
func (h *APIHandler) MudarSenha(w http.ResponseWriter, r *http.Request) {
    h.SetHeaders(w, r)
    var req ReqMudarSenha
    json.NewDecoder(r.Body).Decode(&req)

    err := h.uc.MudarSenha(r.Context(), usecase.MudarSenhaInput{
        CPF: req.CPF, SenhaAtual: req.SenhaAtual, NovaSenha: req.NovaSenha,
    })
    if err != nil {
        json.NewEncoder(w).Encode(RespostaAPI{Sucesso: false, Mensagem: err.Error()})
        return
    }
    json.NewEncoder(w).Encode(RespostaAPI{Sucesso: true, Mensagem: "Senha alterada!"})
}

// PUT /api/dados
func (h *APIHandler) EditarDados(w http.ResponseWriter, r *http.Request) {
    h.SetHeaders(w, r)
    var req ReqEditarDados
    json.NewDecoder(r.Body).Decode(&req)

    err := h.uc.EditarDados(r.Context(), usecase.EditarDadosInput{
        CPF: req.CPF, NovoNome: req.NovoNome, NovoLimite: req.NovoLimite,
    })
    if err != nil {
        json.NewEncoder(w).Encode(RespostaAPI{Sucesso: false, Mensagem: err.Error()})
        return
    }
    json.NewEncoder(w).Encode(RespostaAPI{Sucesso: true, Mensagem: "Dados atualizados!"})
}

