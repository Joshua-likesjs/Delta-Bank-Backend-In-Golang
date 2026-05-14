package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"delta-bank/internal/domain"
	"delta-bank/internal/usecase"

	"github.com/gorilla/mux"
)

// APIHandler recebe requisições HTTP e delega ao BankService.
// FIX: antes dependia de *usecase.Service (concreto);
// agora depende de usecase.BankService (interface) — testável.
type APIHandler struct {
	svc usecase.BankService
}

// NovoAPIHandler cria um handler com serviço injetado via interface.
func NovoAPIHandler(svc usecase.BankService) *APIHandler {
	return &APIHandler{svc: svc}
}

// Health retorna o status da API.
func (h *APIHandler) Health(w http.ResponseWriter, _ *http.Request) {
	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

// POST /api/contas
func (h *APIHandler) CriarConta(w http.ResponseWriter, r *http.Request) {
	var req ReqCriarConta
	if err := decode(r, &req); err != nil {
		respondError(w, domain.NewErro("corpo da requisição inválido", http.StatusBadRequest))
		return
	}

	conta, err := h.svc.CriarConta(r.Context(), usecase.CriarContaInput{
		Nome: req.Nome, CPF: req.CPF, Senha: req.Senha, Saldo: req.Saldo,
	})
	if err != nil {
		respondError(w, err)
		return
	}

	respondCreated(w, "Conta criada!", conta)
}

// POST /api/login
func (h *APIHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req ReqLogin
	if err := decode(r, &req); err != nil {
		respondError(w, domain.NewErro("corpo da requisição inválido", http.StatusBadRequest))
		return
	}

	conta, err := h.svc.Login(r.Context(), usecase.LoginInput{
		CPF: req.CPF, Senha: req.Senha,
	})
	if err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, "Login OK!", conta)
}

// GET /api/saldo/{cpf}
func (h *APIHandler) Saldo(w http.ResponseWriter, r *http.Request) {
	output, err := h.svc.ConsultarSaldo(r.Context(), mux.Vars(r)["cpf"])
	if err != nil {
		respondError(w, err)
		return
	}
	respondOK(w, "", output)
}

// GET /api/extrato/{cpf}
func (h *APIHandler) Extrato(w http.ResponseWriter, r *http.Request) {
	transacoes, saldo, err := h.svc.Extrato(r.Context(), mux.Vars(r)["cpf"])
	if err != nil {
		respondError(w, err)
		return
	}
	respondOK(w, "", map[string]any{
		"transacoes": transacoes,
		"saldo_atual": saldo,
	})
}

// POST /api/pix
func (h *APIHandler) Pix(w http.ResponseWriter, r *http.Request) {
	var req ReqPix
	if err := decode(r, &req); err != nil {
		respondError(w, domain.NewErro("corpo da requisição inválido", http.StatusBadRequest))
		return
	}

	tx, err := h.svc.FazerPix(r.Context(), usecase.PixInput{
		CPFOrigem: req.CPFOrigem, CPFDestino: req.CPFDestino, Valor: req.Valor,
	})
	if err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, "PIX realizado!", tx)
}

// POST /api/depositar
func (h *APIHandler) Depositar(w http.ResponseWriter, r *http.Request) {
	var req ReqDeposito
	if err := decode(r, &req); err != nil {
		respondError(w, domain.NewErro("corpo da requisição inválido", http.StatusBadRequest))
		return
	}

	novoSaldo, err := h.svc.Depositar(r.Context(), usecase.DepositoInput{
		CPF: req.CPF, Valor: req.Valor,
	})
	if err != nil {
		respondError(w, err)
		return
	}

	// FIX: campo nomeado "novo_saldo_centavos" para deixar claro que é saldo, não valor depositado.
	respondOK(w, "Depósito realizado!", map[string]int{"novo_saldo_centavos": novoSaldo})
}

// POST /api/sacar
func (h *APIHandler) Sacar(w http.ResponseWriter, r *http.Request) {
	var req ReqSaque
	if err := decode(r, &req); err != nil {
		respondError(w, domain.NewErro("corpo da requisição inválido", http.StatusBadRequest))
		return
	}

	novoSaldo, err := h.svc.Sacar(r.Context(), usecase.SaqueInput{
		CPF: req.CPF, Valor: req.Valor,
	})
	if err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, "Saque realizado!", map[string]int{"novo_saldo_centavos": novoSaldo})
}

// GET /api/chaves-pix/{cpf}
func (h *APIHandler) ListarChaves(w http.ResponseWriter, r *http.Request) {
	chaves, err := h.svc.ListarChaves(r.Context(), mux.Vars(r)["cpf"])
	if err != nil {
		respondError(w, err)
		return
	}
	respondOK(w, "", chaves)
}

// POST /api/chaves-pix
func (h *APIHandler) AdicionarChave(w http.ResponseWriter, r *http.Request) {
	var req ReqAdicionarChave
	if err := decode(r, &req); err != nil {
		respondError(w, domain.NewErro("corpo da requisição inválido", http.StatusBadRequest))
		return
	}

	chave, err := h.svc.AdicionarChave(r.Context(), usecase.AdicionarChaveInput{
		CPF: req.CPF, Tipo: req.Tipo, Valor: req.Valor,
	})
	if err != nil {
		respondError(w, err)
		return
	}

	respondCreated(w, "Chave adicionada!", chave)
}

// DELETE /api/chaves-pix
func (h *APIHandler) RemoverChave(w http.ResponseWriter, r *http.Request) {
	var req ReqRemoverChave
	if err := decode(r, &req); err != nil {
		respondError(w, domain.NewErro("corpo da requisição inválido", http.StatusBadRequest))
		return
	}

	if err := h.svc.RemoverChave(r.Context(), usecase.RemoverChaveInput{
		CPF: req.CPF, Tipo: req.Tipo, Valor: req.Valor,
	}); err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, "Chave removida!", nil)
}

// PUT /api/senha
func (h *APIHandler) MudarSenha(w http.ResponseWriter, r *http.Request) {
	var req ReqMudarSenha
	if err := decode(r, &req); err != nil {
		respondError(w, domain.NewErro("corpo da requisição inválido", http.StatusBadRequest))
		return
	}

	if err := h.svc.MudarSenha(r.Context(), usecase.MudarSenhaInput{
		CPF: req.CPF, SenhaAtual: req.SenhaAtual, NovaSenha: req.NovaSenha,
	}); err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, "Senha alterada!", nil)
}

// PUT /api/dados
func (h *APIHandler) EditarDados(w http.ResponseWriter, r *http.Request) {
	var req ReqEditarDados
	if err := decode(r, &req); err != nil {
		respondError(w, domain.NewErro("corpo da requisição inválido", http.StatusBadRequest))
		return
	}

	if err := h.svc.EditarDados(r.Context(), usecase.EditarDadosInput{
		CPF: req.CPF, NovoNome: req.NovoNome, NovoLimite: req.NovoLimite,
	}); err != nil {
		respondError(w, err)
		return
	}

	respondOK(w, "Dados atualizados!", nil)
}

// ============ HELPERS ============

// decode desserializa o body JSON e retorna erro se falhar.
func decode(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

// respond serializa payload como JSON com o status informado.
func respond(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		json.NewEncoder(w).Encode(payload) //nolint:errcheck
	}
}

// respondOK envia 200 com envelope de sucesso.
func respondOK(w http.ResponseWriter, msg string, dados any) {
	respond(w, http.StatusOK, RespostaAPI{Sucesso: true, Mensagem: msg, Dados: dados})
}

// respondCreated envia 201 com envelope de sucesso.
func respondCreated(w http.ResponseWriter, msg string, dados any) {
	respond(w, http.StatusCreated, RespostaAPI{Sucesso: true, Mensagem: msg, Dados: dados})
}

// respondError mapeia o erro para o status HTTP correto e nunca vaza
// detalhes internos (erros não-domínio viram 500 genérico).
// FIX: antes todos os erros retornavam HTTP 200 — agora retornam o código correto.
func respondError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	msg := "erro interno do servidor"

	var domErr *domain.ErroDominio
	if errors.As(err, &domErr) {
		code = domErr.Codigo
		msg = domErr.Mensagem
	}

	respond(w, code, RespostaAPI{Sucesso: false, Mensagem: msg})
}