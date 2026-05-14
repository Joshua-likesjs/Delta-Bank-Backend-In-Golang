package usecase

import (
	"context"

	"delta-bank/internal/domain"
)

// ============ INPUTS ============

type CriarContaInput struct {
	Nome  string
	CPF   string
	Senha string
	Saldo float64
}

type LoginInput struct {
	CPF   string
	Senha string
}

type PixInput struct {
	CPFOrigem  string
	CPFDestino string
	Valor      float64
}

type DepositoInput struct {
	CPF   string
	Valor float64
}

type SaqueInput struct {
	CPF   string
	Valor float64
}

type MudarSenhaInput struct {
	CPF        string
	SenhaAtual string
	NovaSenha  string
}

type EditarDadosInput struct {
	CPF        string
	NovoNome   string
	NovoLimite float64
}

type AdicionarChaveInput struct {
	CPF   string
	Tipo  string
	Valor string
}

type RemoverChaveInput struct {
	CPF   string
	Tipo  string
	Valor string
}

// ============ OUTPUTS ============

type SaldoOutput struct {
	Conta     *domain.Conta `json:"Conta"`
	QtdChaves int           `json:"QtdChaves"`
}

// ============ INTERFACE DO SERVIÇO ============

type BankService interface {
	CriarConta(ctx context.Context, input CriarContaInput) (*domain.Conta, error)
	Login(ctx context.Context, input LoginInput) (*domain.Conta, error)
	ConsultarSaldo(ctx context.Context, cpf string) (*SaldoOutput, error)
	Extrato(ctx context.Context, cpf string) ([]*domain.Transacao, int, error)
	FazerPix(ctx context.Context, input PixInput) (*domain.Transacao, error)
	Depositar(ctx context.Context, input DepositoInput) (int, error)
	Sacar(ctx context.Context, input SaqueInput) (int, error)
	ListarChaves(ctx context.Context, cpf string) ([]*domain.ChavePix, error)
	AdicionarChave(ctx context.Context, input AdicionarChaveInput) (*domain.ChavePix, error)
	RemoverChave(ctx context.Context, input RemoverChaveInput) error
	MudarSenha(ctx context.Context, input MudarSenhaInput) error
	EditarDados(ctx context.Context, input EditarDadosInput) error
}