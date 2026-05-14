package repository

import (
	"context"

	"delta-bank/internal/domain"
)

// Repository é a única interface de persistência que o domínio conhece.
// PostgresRepo a implementa; testes podem usar mocks.
// FIX: as 3 interfaces anteriores (ContaRepository, TransacaoRepository,
// ChavePixRepository) nunca eram usadas — o service dependia diretamente
// de *PostgresRepo, quebrando a Clean Architecture por completo.
type Repository interface {
	// — Conta —
	SalvarConta(ctx context.Context, conta *domain.Conta) error
	BuscarContaPorCPF(ctx context.Context, cpf string) (*domain.Conta, error)
	ExisteContaPorCPF(ctx context.Context, cpf string) (bool, error)
	AtualizarSaldo(ctx context.Context, cpf string, novoSaldo int) error
	AtualizarDados(ctx context.Context, cpf string, nome string, limite int) error
	AtualizarSenha(ctx context.Context, cpf string, novoHash string) error

	// — Transação —
	SalvarTransacao(ctx context.Context, t *domain.Transacao) error
	ListarTransacoesPorCPF(ctx context.Context, cpf string) ([]*domain.Transacao, error)
	SomarPixHoje(ctx context.Context, cpf string) (int, error)

	// — Chave PIX —
	SalvarChavePix(ctx context.Context, chave *domain.ChavePix) error
	ListarChavesPorCPF(ctx context.Context, cpf string) ([]*domain.ChavePix, error)
	BuscarChavePorValor(ctx context.Context, valor string) (*domain.ChavePix, error)
	ContarChavesPorCPF(ctx context.Context, cpf string) (int, error)
	DeletarChavePix(ctx context.Context, cpf, tipo, valor string) error

	// — Controle transacional —
	// ExecTx executa fn dentro de uma transação atômica com rollback automático.
	ExecTx(ctx context.Context, fn func(Repository) error) error
}