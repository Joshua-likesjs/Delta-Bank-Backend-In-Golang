package repository

import (
	"context"
	"fmt"

	"delta-bank/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// querier abstrai *pgx.Conn e pgx.Tx, permitindo que PostgresRepo
// opere tanto em conexão direta quanto dentro de uma transação.
type querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// PostgresRepo implementa repositórios com PostgreSQL
type PostgresRepo struct {
	conn *pgx.Conn // conexão raiz — usada apenas para iniciar transações
	db   querier   // querier ativo: pode ser *pgx.Conn ou pgx.Tx
}

// NovoPostgresRepo cria nova instância
func NovoPostgresRepo(db *pgx.Conn) *PostgresRepo {
	return &PostgresRepo{conn: db, db: db}
}

// ExecTransacao executa fn dentro de uma transação atômica.
// Faz rollback automaticamente se fn retornar erro ou ocorrer panic.
func (r *PostgresRepo) ExecTransacao(ctx context.Context, fn func(*PostgresRepo) error) error {
	tx, err := r.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("erro iniciar transacao: %w", err)
	}
	defer tx.Rollback(ctx) // no-op se Commit já foi chamado

	txRepo := &PostgresRepo{conn: r.conn, db: tx}

	if err := fn(txRepo); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("erro commit: %w", err)
	}
	return nil
}

// ============ CONTA ============

func (r *PostgresRepo) Salvar(ctx context.Context, c *domain.Conta) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO contas (cpf, owner, balance, senha, limite_diario, created_at) 
         VALUES ($1, $2, $3, $4, $5, NOW())`,
		c.CPF, c.Nome, c.SaldoCentavos, c.SenhaHash, c.LimiteDiario)
	return err
}

func (r *PostgresRepo) BuscarPorCPF(ctx context.Context, cpf string) (*domain.Conta, error) {
	var c domain.Conta
	err := r.db.QueryRow(ctx,
		"SELECT cpf, owner, balance, senha, limite_diario, created_at FROM contas WHERE cpf=$1",
		cpf).Scan(&c.CPF, &c.Nome, &c.SaldoCentavos, &c.SenhaHash, &c.LimiteDiario, &c.CriadaEm)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *PostgresRepo) VerificarExistePorCPF(ctx context.Context, cpf string) (bool, error) {
	var existe bool
	err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM contas WHERE cpf=$1)", cpf).Scan(&existe)
	return existe, err
}

func (r *PostgresRepo) AtualizarSaldo(ctx context.Context, cpf string, saldo int) error {
	_, err := r.db.Exec(ctx, "UPDATE contas SET balance=$1, updated_at=NOW() WHERE cpf=$2", saldo, cpf)
	return err
}

func (r *PostgresRepo) AtualizarDados(ctx context.Context, cpf string, nome string, limite int) error {
	_, err := r.db.Exec(ctx, "UPDATE contas SET owner=$1, limite_diario=$2, updated_at=NOW() WHERE cpf=$3", nome, limite, cpf)
	return err
}

func (r *PostgresRepo) AtualizarSenha(ctx context.Context, cpf string, hash string) error {
	_, err := r.db.Exec(ctx, "UPDATE contas SET senha=$1, updated_at=NOW() WHERE cpf=$2", hash, cpf)
	return err
}

// ============ TRANSAÇÃO ============

func (r *PostgresRepo) SalvarTransacao(ctx context.Context, t *domain.Transacao) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO transacoes (conta_cpf_remetente, conta_cpf_destinatario, tipo, valor_centavos)
         VALUES ($1, $2, $3, $4)`,
		t.CPFRemetente, t.CPFDestinatario, t.Tipo, t.ValorCentavos)
	return err
}

func (r *PostgresRepo) ListarTransacoesPorCPF(ctx context.Context, cpf string) ([]*domain.Transacao, error) {

	

	rows, err := r.db.Query(ctx,
		`SELECT t.id, t.tipo, t.valor_centavos, t.conta_cpf_remetente, t.conta_cpf_destinatario, t.created_at,
                COALESCE(cr.owner,'Desconhecido') as nome_remetente,
                COALESCE(cd.owner,'Desconhecido') as nome_destinatario
         FROM transacoes t
         LEFT JOIN contas cr ON t.conta_cpf_remetente = cr.cpf
         LEFT JOIN contas cd ON t.conta_cpf_destinatario = cd.cpf
         WHERE t.conta_cpf_remetente=$1 OR t.conta_cpf_destinatario=$1
         ORDER BY t.created_at DESC`, cpf)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transacoes []*domain.Transacao
	for rows.Next() {
		var t domain.Transacao
		rows.Scan(&t.ID, &t.Tipo, &t.ValorCentavos, &t.CPFRemetente, &t.CPFDestinatario, &t.DataHora, &t.NomeRemetente, &t.NomeDestinatario)
		transacoes = append(transacoes, &t)
	}
	return transacoes, nil
}

func (r *PostgresRepo) SomarValorHoje(ctx context.Context, cpf string, tipo string) (int, error) {
	var soma int
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(valor_centavos),0) FROM transacoes 
         WHERE conta_cpf_remetente=$1 AND tipo=$2 AND DATE(created_at)=CURRENT_DATE`,
		cpf, tipo).Scan(&soma)
	return soma, err
}

// ============ CHAVE PIX ============

func (r *PostgresRepo) SalvarChavePix(ctx context.Context, k *domain.ChavePix) error {
	_, err := r.db.Exec(ctx,
		"INSERT INTO chaves_pix (conta_cpf, tipo, valor, created_at) VALUES ($1, $2, $3, NOW())",
		k.CPFConta, k.Tipo, k.Valor)
	return err
}

func (r *PostgresRepo) ListarChavesPorCPF(ctx context.Context, cpf string) ([]*domain.ChavePix, error) {
	rows, err := r.db.Query(ctx,
		// FIX: adicionado conta_cpf ao SELECT para o Scan não ficar com colunas a menos
		"SELECT id, tipo, valor, conta_cpf, created_at FROM chaves_pix WHERE conta_cpf=$1 ORDER BY created_at", cpf)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chaves []*domain.ChavePix
	for rows.Next() {
		var k domain.ChavePix
		rows.Scan(&k.ID, &k.Tipo, &k.Valor, &k.CPFConta, &k.CriadaEm)
		chaves = append(chaves, &k)
	}
	return chaves, nil
}

func (r *PostgresRepo) BuscarChavePorValor(ctx context.Context, valor string) (*domain.ChavePix, error) {
	var k domain.ChavePix
	err := r.db.QueryRow(ctx,
		`SELECT ck.id, ck.tipo, ck.valor, ck.conta_cpf, ck.created_at 
         FROM chaves_pix ck JOIN contas c ON ck.conta_cpf = c.cpf WHERE ck.valor=$1`, valor).
		Scan(&k.ID, &k.Tipo, &k.Valor, &k.CPFConta, &k.CriadaEm)
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *PostgresRepo) ExisteChaveParaConta(ctx context.Context, cpf string, tipo string) (bool, error) {
	var existe bool
	err := r.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM chaves_pix WHERE conta_cpf=$1 AND tipo=$2)", cpf, tipo).Scan(&existe)
	return existe, err
}

func (r *PostgresRepo) DeletarChavePix(ctx context.Context, cpf string, tipo string, valor string) error {
	_, err := r.db.Exec(ctx,
		"DELETE FROM chaves_pix WHERE conta_cpf=$1 AND tipo=$2 AND valor=$3", cpf, tipo, valor)
	return err
}

func (r *PostgresRepo) ContarChavesPorCPF(ctx context.Context, cpf string) (int, error) {
	var qtd int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM chaves_pix WHERE conta_cpf=$1", cpf).Scan(&qtd)
	return qtd, err
}