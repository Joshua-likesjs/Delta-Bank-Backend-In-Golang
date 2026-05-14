package repository

import (
	"context"
	"errors"
	"fmt"

	"delta-bank/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// querier abstrai *pgxpool.Pool e pgx.Tx, permitindo que o repo
// opere tanto em conexão direta quanto dentro de uma transação.
type querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// PostgresRepo implementa Repository usando PostgreSQL + pgxpool.
// FIX: substituído pgx.Conn (single connection) por pgxpool.Pool
// para suportar concorrência real em produção.
type PostgresRepo struct {
	pool *pgxpool.Pool // usado para abrir transações
	db   querier       // querier ativo: pool ou pgx.Tx
}

// NovoPostgresRepo cria uma instância com pool de conexões.
func NovoPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool, db: pool}
}

// ExecTx executa fn dentro de uma transação atômica.
// Faz rollback automaticamente se fn retornar erro ou ocorrer panic.
// FIX: a versão anterior usava pgx.Conn.Begin; agora usa pgxpool.Pool.Begin.
func (r *PostgresRepo) ExecTx(ctx context.Context, fn func(Repository) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("erro ao iniciar transação: %w", err)
	}
	defer tx.Rollback(ctx) // no-op se Commit já foi chamado

	txRepo := &PostgresRepo{pool: r.pool, db: tx}

	if err := fn(txRepo); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("erro ao fazer commit: %w", err)
	}
	return nil
}

// ============ CONTA ============

func (r *PostgresRepo) SalvarConta(ctx context.Context, c *domain.Conta) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO contas (cpf, owner, balance, senha, limite_diario, created_at)
         VALUES ($1, $2, $3, $4, $5, NOW())`,
		c.CPF, c.Nome, c.SaldoCentavos, c.SenhaHash, c.LimiteDiario,
	)
	return err
}

func (r *PostgresRepo) BuscarContaPorCPF(ctx context.Context, cpf string) (*domain.Conta, error) {
	var c domain.Conta
	err := r.db.QueryRow(ctx,
		`SELECT cpf, owner, balance, senha, limite_diario, created_at
         FROM contas WHERE cpf = $1`, cpf,
	).Scan(&c.CPF, &c.Nome, &c.SaldoCentavos, &c.SenhaHash, &c.LimiteDiario, &c.CriadaEm)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrContaNaoEncontrada
		}
		return nil, fmt.Errorf("erro ao buscar conta: %w", err)
	}
	return &c, nil
}

func (r *PostgresRepo) ExisteContaPorCPF(ctx context.Context, cpf string) (bool, error) {
	var existe bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM contas WHERE cpf = $1)`, cpf,
	).Scan(&existe)
	return existe, err
}

func (r *PostgresRepo) AtualizarSaldo(ctx context.Context, cpf string, novoSaldo int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE contas SET balance = $1, updated_at = NOW() WHERE cpf = $2`,
		novoSaldo, cpf,
	)
	return err
}

func (r *PostgresRepo) AtualizarDados(ctx context.Context, cpf, nome string, limite int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE contas SET owner = $1, limite_diario = $2, updated_at = NOW() WHERE cpf = $3`,
		nome, limite, cpf,
	)
	return err
}

func (r *PostgresRepo) AtualizarSenha(ctx context.Context, cpf, hash string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE contas SET senha = $1, updated_at = NOW() WHERE cpf = $2`,
		hash, cpf,
	)
	return err
}

// ============ TRANSAÇÃO ============

func (r *PostgresRepo) SalvarTransacao(ctx context.Context, t *domain.Transacao) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO transacoes (conta_cpf_remetente, conta_cpf_destinatario, tipo, valor_centavos)
         VALUES ($1, $2, $3, $4)`,
		t.CPFRemetente, t.CPFDestinatario, t.Tipo, t.ValorCentavos,
	)
	return err
}

func (r *PostgresRepo) ListarTransacoesPorCPF(ctx context.Context, cpf string) ([]*domain.Transacao, error) {
	rows, err := r.db.Query(ctx,
		`SELECT
             t.id, t.tipo, t.valor_centavos,
             t.conta_cpf_remetente, t.conta_cpf_destinatario, t.created_at,
             COALESCE(cr.owner, 'Desconhecido') AS nome_remetente,
             COALESCE(cd.owner, 'Desconhecido') AS nome_destinatario
         FROM transacoes t
         LEFT JOIN contas cr ON t.conta_cpf_remetente    = cr.cpf
         LEFT JOIN contas cd ON t.conta_cpf_destinatario = cd.cpf
         WHERE t.conta_cpf_remetente = $1 OR t.conta_cpf_destinatario = $1
         ORDER BY t.created_at DESC`, cpf,
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar transações: %w", err)
	}
	defer rows.Close()

	var transacoes []*domain.Transacao
	for rows.Next() {
		var t domain.Transacao
		if err := rows.Scan(
			&t.ID, &t.Tipo, &t.ValorCentavos,
			&t.CPFRemetente, &t.CPFDestinatario, &t.DataHora,
			&t.NomeRemetente, &t.NomeDestinatario,
		); err != nil {
			return nil, fmt.Errorf("erro ao ler transação: %w", err)
		}
		transacoes = append(transacoes, &t)
	}
	return transacoes, rows.Err()
}

// SomarPixHoje soma o valor total de PIX enviados pelo CPF no dia corrente.
// FIX: antes recebia o tipo como parâmetro — agora é específico para PIX,
// tornando o contrato mais claro.
func (r *PostgresRepo) SomarPixHoje(ctx context.Context, cpf string) (int, error) {
	var soma int
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(valor_centavos), 0)
         FROM transacoes
         WHERE conta_cpf_remetente = $1
           AND tipo = 'TRANSACAO_PIX'
           AND DATE(created_at) = CURRENT_DATE`, cpf,
	).Scan(&soma)
	return soma, err
}

// ============ CHAVE PIX ============

func (r *PostgresRepo) SalvarChavePix(ctx context.Context, k *domain.ChavePix) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO chaves_pix (conta_cpf, tipo, valor, created_at) VALUES ($1, $2, $3, NOW())`,
		k.CPFConta, k.Tipo, k.Valor,
	)
	return err
}

func (r *PostgresRepo) ListarChavesPorCPF(ctx context.Context, cpf string) ([]*domain.ChavePix, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, tipo, valor, conta_cpf, created_at
         FROM chaves_pix WHERE conta_cpf = $1 ORDER BY created_at`, cpf,
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar chaves: %w", err)
	}
	defer rows.Close()

	var chaves []*domain.ChavePix
	for rows.Next() {
		var k domain.ChavePix
		if err := rows.Scan(&k.ID, &k.Tipo, &k.Valor, &k.CPFConta, &k.CriadaEm); err != nil {
			return nil, fmt.Errorf("erro ao ler chave: %w", err)
		}
		chaves = append(chaves, &k)
	}
	return chaves, rows.Err()
}

func (r *PostgresRepo) BuscarChavePorValor(ctx context.Context, valor string) (*domain.ChavePix, error) {
	var k domain.ChavePix
	err := r.db.QueryRow(ctx,
		`SELECT ck.id, ck.tipo, ck.valor, ck.conta_cpf, ck.created_at
         FROM chaves_pix ck
         JOIN contas c ON ck.conta_cpf = c.cpf
         WHERE ck.valor = $1`, valor,
	).Scan(&k.ID, &k.Tipo, &k.Valor, &k.CPFConta, &k.CriadaEm)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // chave não encontrada não é erro fatal
		}
		return nil, fmt.Errorf("erro ao buscar chave: %w", err)
	}
	return &k, nil
}

func (r *PostgresRepo) ContarChavesPorCPF(ctx context.Context, cpf string) (int, error) {
	var qtd int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM chaves_pix WHERE conta_cpf = $1`, cpf,
	).Scan(&qtd)
	return qtd, err
}

func (r *PostgresRepo) DeletarChavePix(ctx context.Context, cpf, tipo, valor string) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM chaves_pix WHERE conta_cpf = $1 AND tipo = $2 AND valor = $3`,
		cpf, tipo, valor,
	)
	if err != nil {
		return fmt.Errorf("erro ao deletar chave: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrChaveNaoEncontrada
	}
	return nil
}