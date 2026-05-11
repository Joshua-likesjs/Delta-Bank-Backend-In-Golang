package repository

import (
    "context"
    
    "delta-bank/internal/domain"
)

// ContaRepository define operações de banco para contas
type ContaRepository interface {
    Salvar(ctx context.Context, conta *domain.Conta) error
    BuscarPorCPF(ctx context.Context, cpf string) (*domain.Conta, error)
    VerificarExistePorCPF(ctx context.Context, cpf string) (bool, error)
    AtualizarSaldo(ctx context.Context, cpf string, novoSaldo int) error
    AtualizarDados(ctx context.Context, cpf string, nome string, limite int) error
    AtualizarSenha(ctx context.Context, cpf string, novoHash string) error
}

// TransacaoRepository define operações de banco para transações
type TransacaoRepository interface {
    Salvar(ctx context.Context, tx *domain.Transacao) error
    ListarPorCPF(ctx context.Context, cpf string) ([]*domain.Transacao, error)
    SomarValorHoje(ctx context.Context, cpf string, tipo string) (int, error)
}

// ChavePixRepository define operações de banco para chaves PIX
type ChavePixRepository interface {
    Salvar(ctx context.Context, chave *domain.ChavePix) error
    ListarPorCPF(ctx context.Context, cpf string) ([]*domain.ChavePix, error)
    BuscarPorValor(ctx context.Context, valor string) (*domain.ChavePix, error)
    ExisteParaConta(ctx context.Context, cpf string, tipo string) (bool, error)
    Deletar(ctx context.Context, cpf string, tipo string, valor string) error
    ContarPorCPF(ctx context.Context, cpf string) (int, error)
}