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
    Tipo  string // CPF, Email, Telefone, Aleatoria
    Valor string
}

type RemoverChaveInput struct {
    CPF   string
    Tipo  string
    Valor string
}

// ============ OUTPUTS ============

type SaldoOutput struct {
    Conta       *domain.Conta
    QtdChaves   int
}

// ============ INTERFACES DOS USE CASES ============

type CriarContaUC interface {
    Executar(ctx context.Context, input CriarContaInput) (*domain.Conta, error)
}

type LoginUC interface {
    Executar(ctx context.Context, input LoginInput) (*domain.Conta, error)
}

type ConsultarSaldoUC interface {
    Executar(ctx context.Context, cpf string) (*SaldoOutput, error)
}

type ExtratoUC interface {
    Executar(ctx context.Context, cpf string) ([]*domain.Transacao, int, error)
}

type FazerPixUC interface {
    Executar(ctx context.Context, input PixInput) (*domain.Transacao, error)
}

type DepositarUC interface {
    Executar(ctx context.Context, input DepositoInput) (int, error)
}

type SacarUC interface {
    Executar(ctx context.Context, input SaqueInput) (int, error)
}

type ListarChavesUC interface {
    Executar(ctx context.Context, cpf string) ([]*domain.ChavePix, error)
}

type AdicionarChaveUC interface {
    Executar(ctx context.Context, input AdicionarChaveInput) (*domain.ChavePix, error)
}

type RemoverChaveUC interface {
    Executar(ctx context.Context, input RemoverChaveInput) error
}

type MudarSenhaUC interface {
    Executar(ctx context.Context, input MudarSenhaInput) error
}

type EditarDadosUC interface {
    Executar(ctx context.Context, input EditarDadosInput) error
}