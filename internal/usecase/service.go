package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"delta-bank/internal/domain"
	"delta-bank/internal/repository"
	"delta-bank/internal/validation"

	"github.com/jackc/pgx/v5"
)

// Service implementa todos os use cases do sistema
type Service struct {
	repo *repository.PostgresRepo
}

// NovoServico cria novo serviço com repositório injetado
func NovoServico(repo *repository.PostgresRepo) *Service {
	return &Service{repo: repo}
}

// ============ CRIAR CONTA ============
func (s *Service) CriarConta(ctx context.Context, input CriarContaInput) (*domain.Conta, error) {
	if len(input.Nome) < 3 {
		return nil, domain.ErrNomeCurto
	}

	cpf, ok := validation.ValidadorCPF(input.CPF)
	if !ok {
		return nil, domain.ErrCPFINvalido
	}
	if !validation.ValidadorSenha(input.Senha) {
		return nil, domain.ErrSenhaCurta
	}
	if input.Saldo < 10 {
		return nil, domain.ErrValorInvalido
	}

	existe, _ := s.repo.VerificarExistePorCPF(ctx, cpf)
	if existe {
		return nil, domain.ErrCPFJaCadastrado
	}

	hash, err := validation.HashSenha(input.Senha)
	if err != nil {
		return nil, fmt.Errorf("erro senha: %w", err)
	}

	conta := &domain.Conta{
		CPF:           cpf,
		Nome:          input.Nome,
		SaldoCentavos: int(input.Saldo * 100),
		SenhaHash:     hash,
		LimiteDiario:  5000000,
	}

	if err := s.repo.Salvar(ctx, conta); err != nil {
		return nil, fmt.Errorf("erro salvar: %w", err)
	}
	return conta, nil
}

// ============ LOGIN ============
func (s *Service) Login(ctx context.Context, input LoginInput) (*domain.Conta, error) {
	cpf, ok := validation.ValidadorCPF(input.CPF)
	if !ok {
		return nil, domain.ErrCPFINvalido
	}

	conta, err := s.repo.BuscarPorCPF(ctx, cpf)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrContaNaoEncontrada
		}
		return nil, fmt.Errorf("erro buscar: %w", err)
	}

	if !validation.VerificarSenha(input.Senha, conta.SenhaHash) {
		return nil, domain.ErrSenhaIncorreta
	}

	conta.SenhaHash = "" // NUNCA retornar senha!
	return conta, nil
}

// ============ CONSULTAR SALDO ============
func (s *Service) ConsultarSaldo(ctx context.Context, cpf string) (*SaldoOutput, error) {
	conta, err := s.repo.BuscarPorCPF(ctx, cpf)
	if err != nil {
		return nil, domain.ErrContaNaoEncontrada
	}

	qtd, _ := s.repo.ContarChavesPorCPF(ctx, cpf)

	conta.SenhaHash = ""
	return &SaldoOutput{Conta: conta, QtdChaves: qtd}, nil
}

// ============ EXTRATO ============
func (s *Service) Extrato(ctx context.Context, cpf string) ([]*domain.Transacao, int, error) {
	conta, err := s.repo.BuscarPorCPF(ctx, cpf)
	if err != nil {
		return nil, 0, domain.ErrContaNaoEncontrada
	}

	transacoes, err := s.repo.ListarTransacoesPorCPF(ctx, cpf)
	if err != nil {
		return nil, 0, fmt.Errorf("erro extrato: %w", err)
	}

	return transacoes, conta.SaldoCentavos, nil
}

// ============ FAZER PIX ============
// FIX 1: todas as escritas agora ocorrem dentro de uma transação atômica
// para evitar inconsistência de saldo em caso de falha parcial.
func (s *Service) FazerPix(ctx context.Context, input PixInput) (*domain.Transacao, error) {
	// FIX 6: condição redundante removida (< 1 já cobre <= 0 para float)
	if input.Valor < 1 {
		return nil, domain.ErrValorInvalido
	}

	valorCent := int(input.Valor * 100)

	origem, err := s.repo.BuscarPorCPF(ctx, input.CPFOrigem)
	if err != nil {
		return nil, domain.ErrContaNaoEncontrada
	}
	destino, err := s.repo.BuscarPorCPF(ctx, input.CPFDestino)
	if err != nil {
		return nil, domain.ErrDestinatarioNaoEncontrado
	}
	if input.CPFOrigem == input.CPFDestino {
		return nil, domain.ErrAutoTransferencia
	}

	if origem.SaldoCentavos < valorCent {
		return nil, domain.ErrSaldoInsuficiente
	}

	usadoHoje, _ := s.repo.SomarValorHoje(ctx, input.CPFOrigem, "TRANSACAO_PIX")
	if (usadoHoje + valorCent) > origem.LimiteDiario {
		return nil, domain.ErrLimiteExcedido
	}

	tx := &domain.Transacao{
		Tipo:             "TRANSACAO_PIX",
		ValorCentavos:    valorCent,
		CPFRemetente:     input.CPFOrigem,
		CPFDestinatario:  input.CPFDestino,
		NomeRemetente:    origem.Nome,
		NomeDestinatario: destino.Nome,
		DataHora:         time.Now(),
	}

	// FIX 1 + 4: executa as três escritas em transação atômica e trata erros
	if err := s.repo.ExecTransacao(ctx, func(r *repository.PostgresRepo) error {
		if err := r.AtualizarSaldo(ctx, input.CPFOrigem, origem.SaldoCentavos-valorCent); err != nil {
			return fmt.Errorf("erro debitar origem: %w", err)
		}
		if err := r.AtualizarSaldo(ctx, input.CPFDestino, destino.SaldoCentavos+valorCent); err != nil {
			return fmt.Errorf("erro creditar destino: %w", err)
		}
		if err := r.SalvarTransacao(ctx, tx); err != nil {
			return fmt.Errorf("erro salvar transacao: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return tx, nil
}

// ============ DEPOSITO ============
func (s *Service) Depositar(ctx context.Context, input DepositoInput) (int, error) {
	if input.Valor <= 0 || input.Valor > 10000 {
		return 0, domain.ErrValorInvalido
	}

	cpf, ok := validation.ValidadorCPF(input.CPF)
    if !ok { return 0, domain.ErrCPFINvalido }
    input.CPF = cpf

	valorCent := int(input.Valor * 100)

	conta, err := s.repo.BuscarPorCPF(ctx, input.CPF)
	if err != nil {
		// FIX 3: era ErrExtrato, corrigido para ErrContaNaoEncontrada
		return 0, domain.ErrContaNaoEncontrada
	}

	// FIX 4: erros de escrita agora são tratados
	if err := s.repo.AtualizarSaldo(ctx, input.CPF, conta.SaldoCentavos+valorCent); err != nil {
		return 0, fmt.Errorf("erro atualizar saldo: %w", err)
	}
	if err := s.repo.SalvarTransacao(ctx, &domain.Transacao{
		Tipo:            "DEPOSITO",
		ValorCentavos:   valorCent,
		CPFRemetente:    input.CPF,
		CPFDestinatario: input.CPF,
		DataHora:        time.Now(),
	}); err != nil {
		return 0, fmt.Errorf("erro salvar transacao: %w", err)
	}

	return conta.SaldoCentavos + valorCent, nil
}

// ============ SAQUE ============
func (s *Service) Sacar(ctx context.Context, input SaqueInput) (int, error) {
	if input.Valor <= 0 || input.Valor > 2000 {
		return 0, domain.ErrValorInvalido
	}
	if int(input.Valor*100)%1000 != 0 {
		return 0, domain.ErrValorInvalido // múltiplo de 10
	}

	valorCent := int(input.Valor * 100)

	// FIX 2: erro tratado corretamente, sem risco de nil panic
	conta, err := s.repo.BuscarPorCPF(ctx, input.CPF)
	if err != nil {
		return 0, domain.ErrContaNaoEncontrada
	}

	if conta.SaldoCentavos < valorCent {
		return 0, domain.ErrSaldoInsuficiente
	}

	// FIX 4: erros de escrita agora são tratados
	if err := s.repo.AtualizarSaldo(ctx, input.CPF, conta.SaldoCentavos-valorCent); err != nil {
		return 0, fmt.Errorf("erro atualizar saldo: %w", err)
	}
	if err := s.repo.SalvarTransacao(ctx, &domain.Transacao{
		Tipo:            "SAQUE",
		ValorCentavos:   valorCent,
		CPFRemetente:    input.CPF,
		CPFDestinatario: input.CPF,
		DataHora:        time.Now(),
	}); err != nil {
		return 0, fmt.Errorf("erro salvar transacao: %w", err)
	}

	return conta.SaldoCentavos - valorCent, nil
}

// ============ CHAVES PIX ============
func (s *Service) ListarChaves(ctx context.Context, cpf string) ([]*domain.ChavePix, error) {
	return s.repo.ListarChavesPorCPF(ctx, cpf)
}

func (s *Service) AdicionarChave(ctx context.Context, input AdicionarChaveInput) (*domain.ChavePix, error) {

	// adiciona essa validação no início
	cpf, ok := validation.ValidadorCPF(input.CPF)
	if !ok {
		return nil, domain.ErrCPFINvalido
	}
	input.CPF = cpf

	_, err := s.repo.BuscarPorCPF(ctx, input.CPF)
	if err != nil {
		return nil, domain.ErrContaNaoEncontrada
	}

	qtd, _ := s.repo.ContarChavesPorCPF(ctx, input.CPF)
	if qtd >= 5 {
		return nil, domain.ErrLimiteChavesAtingido
	}

	existe, _ := s.repo.BuscarChavePorValor(ctx, input.Valor)
	if existe != nil {
		return nil, domain.ErrChaveJaExistente
	}

	chave := &domain.ChavePix{CPFConta: input.CPF, Tipo: input.Tipo, Valor: input.Valor}
	if err := s.repo.SalvarChavePix(ctx, chave); err != nil {
		return nil, fmt.Errorf("erro salvar chave: %w", err)
	}
	return chave, nil
}

func (s *Service) RemoverChave(ctx context.Context, input RemoverChaveInput) error {
	err := s.repo.DeletarChavePix(ctx, input.CPF, input.Tipo, input.Valor)
	if err != nil {
		return fmt.Errorf("erro remover: %w", err)
	}
	return nil
}

// ============ MUDAR SENHA ============
func (s *Service) MudarSenha(ctx context.Context, input MudarSenhaInput) error {
	conta, err := s.repo.BuscarPorCPF(ctx, input.CPF)
	if err != nil {
		return domain.ErrContaNaoEncontrada
	}
	if !validation.VerificarSenha(input.SenhaAtual, conta.SenhaHash) {
		return domain.ErrSenhaIncorreta
	}
	if input.NovaSenha == input.SenhaAtual {
		return domain.ErrValorInvalido
	}
	if !validation.ValidadorSenha(input.NovaSenha) {
		return domain.ErrSenhaCurta
	}

	hash, err := validation.HashSenha(input.NovaSenha)
	if err != nil {
		return fmt.Errorf("erro hash: %w", err)
	}
	return s.repo.AtualizarSenha(ctx, input.CPF, hash)
}

// ============ EDITAR DADOS ============
// FIX 5: retorna erro explícito quando nenhum campo válido foi fornecido
func (s *Service) EditarDados(ctx context.Context, input EditarDadosInput) error {
	if len(input.NovoNome) >= 3 {
		return s.repo.AtualizarDados(ctx, input.CPF, input.NovoNome, 0)
	}
	if input.NovoLimite >= 100 && input.NovoLimite <= 50000 {
		return s.repo.AtualizarDados(ctx, input.CPF, "", int(input.NovoLimite*100))
	}
	return domain.ErrValorInvalido
}
