package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"delta-bank/internal/domain"
	"delta-bank/internal/repository"
	"delta-bank/internal/validation"
)

// Service implementa BankService.
// FIX: antes dependia de *repository.PostgresRepo (concreto);
// agora depende de repository.Repository (interface) — testável e desacoplado.
type Service struct {
	repo repository.Repository
}

// NovoServico cria um Service com repositório injetado via interface.
func NovoServico(repo repository.Repository) *Service {
	return &Service{repo: repo}
}

// Garante em compile-time que *Service implementa BankService.
var _ BankService = (*Service)(nil)

// ============ CRIAR CONTA ============

func (s *Service) CriarConta(ctx context.Context, input CriarContaInput) (*domain.Conta, error) {
	if len(strings.TrimSpace(input.Nome)) < 3 {
		return nil, domain.ErrNomeCurto
	}

	cpf, ok := validation.ValidadorCPF(input.CPF)
	if !ok {
		return nil, domain.ErrCPFINvalido
	}
	if !validation.ValidadorSenha(input.Senha) {
		return nil, domain.ErrSenhaCurta
	}

	existe, err := s.repo.ExisteContaPorCPF(ctx, cpf)
	if err != nil {
		return nil, fmt.Errorf("erro ao verificar CPF: %w", err)
	}
	if existe {
		return nil, domain.ErrCPFJaCadastrado
	}

	hash, err := validation.HashSenha(input.Senha)
	if err != nil {
		return nil, fmt.Errorf("erro ao gerar hash: %w", err)
	}

	conta := &domain.Conta{
		CPF:           cpf,
		Nome:          strings.TrimSpace(input.Nome),
		SaldoCentavos: 0,
		SenhaHash:     hash,
		LimiteDiario:  5_000_000, // R$ 50.000,00
	}

	if err := s.repo.SalvarConta(ctx, conta); err != nil {
		return nil, fmt.Errorf("erro ao salvar conta: %w", err)
	}

	conta.SenhaHash = ""
	return conta, nil
}

// ============ LOGIN ============

func (s *Service) Login(ctx context.Context, input LoginInput) (*domain.Conta, error) {
	cpf, ok := validation.ValidadorCPF(input.CPF)
	if !ok {
		return nil, domain.ErrCPFINvalido
	}

	conta, err := s.repo.BuscarContaPorCPF(ctx, cpf)
	if err != nil {
		if errors.Is(err, domain.ErrContaNaoEncontrada) {
			return nil, domain.ErrContaNaoEncontrada
		}
		return nil, fmt.Errorf("erro ao buscar conta: %w", err)
	}

	if !validation.VerificarSenha(input.Senha, conta.SenhaHash) {
		return nil, domain.ErrSenhaIncorreta
	}

	conta.SenhaHash = "" // nunca retornar o hash!
	return conta, nil
}

// ============ CONSULTAR SALDO ============

func (s *Service) ConsultarSaldo(ctx context.Context, cpf string) (*SaldoOutput, error) {
	cpf, ok := validation.ValidadorCPF(cpf)
	if !ok {
		return nil, domain.ErrCPFINvalido
	}

	conta, err := s.repo.BuscarContaPorCPF(ctx, cpf)
	if err != nil {
		return nil, err
	}

	qtd, err := s.repo.ContarChavesPorCPF(ctx, cpf)
	if err != nil {
		return nil, fmt.Errorf("erro ao contar chaves: %w", err)
	}

	conta.SenhaHash = ""
	return &SaldoOutput{Conta: conta, QtdChaves: qtd}, nil
}

// ============ EXTRATO ============

func (s *Service) Extrato(ctx context.Context, cpf string) ([]*domain.Transacao, int, error) {
	cpf, ok := validation.ValidadorCPF(cpf)
	if !ok {
		return nil, 0, domain.ErrCPFINvalido
	}

	conta, err := s.repo.BuscarContaPorCPF(ctx, cpf)
	if err != nil {
		return nil, 0, err
	}

	transacoes, err := s.repo.ListarTransacoesPorCPF(ctx, cpf)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao buscar extrato: %w", err)
	}

	return transacoes, conta.SaldoCentavos, nil
}

// ============ FAZER PIX ============

func (s *Service) FazerPix(ctx context.Context, input PixInput) (*domain.Transacao, error) {
	if input.Valor < 1 {
		return nil, domain.ErrValorInvalido
	}

	cpfOrigem, ok := validation.ValidadorCPF(input.CPFOrigem)
	if !ok {
		return nil, domain.ErrCPFINvalido
	}
	cpfDestino, ok := validation.ValidadorCPF(input.CPFDestino)
	if !ok {
		return nil, domain.ErrDestinatarioNaoEncontrado
	}

	if cpfOrigem == cpfDestino {
		return nil, domain.ErrAutoTransferencia
	}

	valorCent := int(input.Valor * 100)

	origem, err := s.repo.BuscarContaPorCPF(ctx, cpfOrigem)
	if err != nil {
		return nil, err
	}

	destino, err := s.repo.BuscarContaPorCPF(ctx, cpfDestino)
	if err != nil {
		if errors.Is(err, domain.ErrContaNaoEncontrada) {
			return nil, domain.ErrDestinatarioNaoEncontrado
		}
		return nil, err
	}

	if origem.SaldoCentavos < valorCent {
		return nil, domain.ErrSaldoInsuficiente
	}

	usadoHoje, err := s.repo.SomarPixHoje(ctx, cpfOrigem)
	if err != nil {
		return nil, fmt.Errorf("erro ao verificar limite: %w", err)
	}
	if usadoHoje+valorCent > origem.LimiteDiario {
		return nil, domain.ErrLimiteExcedido
	}

	tx := &domain.Transacao{
		Tipo:             "TRANSACAO_PIX",
		ValorCentavos:    valorCent,
		CPFRemetente:     cpfOrigem,
		CPFDestinatario:  cpfDestino,
		NomeRemetente:    origem.Nome,
		NomeDestinatario: destino.Nome,
		DataHora:         time.Now(),
	}

	// FIX: as 3 escritas (debitar, creditar, registrar) ocorrem numa única
	// transação atômica — antes só o PIX tinha isso; agora é garantido.
	if err := s.repo.ExecTx(ctx, func(r repository.Repository) error {
		if err := r.AtualizarSaldo(ctx, cpfOrigem, origem.SaldoCentavos-valorCent); err != nil {
			return fmt.Errorf("erro ao debitar origem: %w", err)
		}
		if err := r.AtualizarSaldo(ctx, cpfDestino, destino.SaldoCentavos+valorCent); err != nil {
			return fmt.Errorf("erro ao creditar destino: %w", err)
		}
		return r.SalvarTransacao(ctx, tx)
	}); err != nil {
		return nil, err
	}

	return tx, nil
}

// ============ DEPÓSITO ============

func (s *Service) Depositar(ctx context.Context, input DepositoInput) (int, error) {
	if input.Valor < 0.01 || input.Valor > 10_000 {
		return 0, domain.ErrValorInvalido
	}

	cpf, ok := validation.ValidadorCPF(input.CPF)
	if !ok {
		return 0, domain.ErrCPFINvalido
	}

	valorCent := int(input.Valor * 100)

	conta, err := s.repo.BuscarContaPorCPF(ctx, cpf)
	if err != nil {
		return 0, err
	}

	novoSaldo := conta.SaldoCentavos + valorCent

	// FIX: operação agora é atômica (antes: duas escritas sem transação)
	if err := s.repo.ExecTx(ctx, func(r repository.Repository) error {
		if err := r.AtualizarSaldo(ctx, cpf, novoSaldo); err != nil {
			return fmt.Errorf("erro ao atualizar saldo: %w", err)
		}
		return r.SalvarTransacao(ctx, &domain.Transacao{
			Tipo:            "DEPOSITO",
			ValorCentavos:   valorCent,
			CPFRemetente:    cpf,
			CPFDestinatario: cpf,
			DataHora:        time.Now(),
		})
	}); err != nil {
		return 0, err
	}

	// FIX: antes retornava valorCent (valor depositado), não o saldo novo.
	return novoSaldo, nil
}

// ============ SAQUE ============

func (s *Service) Sacar(ctx context.Context, input SaqueInput) (int, error) {
	if input.Valor < 0.01 || input.Valor > 2_000 {
		return 0, domain.ErrValorInvalido
	}

	valorCent := int(input.Valor * 100)
	if valorCent%1000 != 0 {
		return 0, domain.ErrValorInvalido // deve ser múltiplo de R$ 10,00
	}

	cpf, ok := validation.ValidadorCPF(input.CPF)
	if !ok {
		return 0, domain.ErrCPFINvalido
	}

	conta, err := s.repo.BuscarContaPorCPF(ctx, cpf)
	if err != nil {
		return 0, err
	}

	if conta.SaldoCentavos < valorCent {
		return 0, domain.ErrSaldoInsuficiente
	}

	novoSaldo := conta.SaldoCentavos - valorCent

	// FIX: operação agora é atômica
	if err := s.repo.ExecTx(ctx, func(r repository.Repository) error {
		if err := r.AtualizarSaldo(ctx, cpf, novoSaldo); err != nil {
			return fmt.Errorf("erro ao atualizar saldo: %w", err)
		}
		return r.SalvarTransacao(ctx, &domain.Transacao{
			Tipo:            "SAQUE",
			ValorCentavos:   valorCent,
			CPFRemetente:    cpf,
			CPFDestinatario: cpf,
			DataHora:        time.Now(),
		})
	}); err != nil {
		return 0, err
	}

	// FIX: antes retornava valorCent (valor sacado), não o saldo novo.
	return novoSaldo, nil
}

// ============ CHAVES PIX ============

func (s *Service) ListarChaves(ctx context.Context, cpf string) ([]*domain.ChavePix, error) {
	cpf, ok := validation.ValidadorCPF(cpf)
	if !ok {
		return nil, domain.ErrCPFINvalido
	}
	return s.repo.ListarChavesPorCPF(ctx, cpf)
}

func (s *Service) AdicionarChave(ctx context.Context, input AdicionarChaveInput) (*domain.ChavePix, error) {
	cpf, ok := validation.ValidadorCPF(input.CPF)
	if !ok {
		return nil, domain.ErrCPFINvalido
	}

	qtd, err := s.repo.ContarChavesPorCPF(ctx, cpf)
	if err != nil {
		return nil, fmt.Errorf("erro ao contar chaves: %w", err)
	}
	if qtd >= 5 {
		return nil, domain.ErrLimiteChavesAtingido
	}

	tipo := strings.ToUpper(input.Tipo)
	valor := input.Valor

	switch tipo {
	case "CPF":
		v, ok := validation.ValidadorCPF(valor)
		if !ok {
			return nil, domain.ErrCPFINvalido
		}
		valor = v
	case "EMAIL":
		v, ok := validation.ValidadorEmail(valor)
		if !ok {
			return nil, domain.ErrValorInvalido
		}
		valor = v
	case "TELEFONE":
		v, ok := validation.ValidadorTelefone(valor)
		if !ok {
			return nil, domain.ErrValorInvalido
		}
		valor = v
	case "ALEATORIA":
		valor = validation.GerarChaveAleatoria()
	default:
		return nil, domain.ErrValorInvalido
	}

	existente, err := s.repo.BuscarChavePorValor(ctx, valor)
	if err != nil {
		return nil, fmt.Errorf("erro ao verificar chave: %w", err)
	}
	if existente != nil {
		return nil, domain.ErrChaveJaExistente
	}

	chave := &domain.ChavePix{CPFConta: cpf, Tipo: tipo, Valor: valor}
	if err := s.repo.SalvarChavePix(ctx, chave); err != nil {
		return nil, fmt.Errorf("erro ao salvar chave: %w", err)
	}
	return chave, nil
}

func (s *Service) RemoverChave(ctx context.Context, input RemoverChaveInput) error {
	cpf, ok := validation.ValidadorCPF(input.CPF)
	if !ok {
		return domain.ErrCPFINvalido
	}

	tipo := strings.ToUpper(input.Tipo)
	valor := input.Valor

	switch tipo {
	case "CPF":
		v, ok := validation.ValidadorCPF(valor)
		if !ok {
			return domain.ErrCPFINvalido
		}
		valor = v
	case "EMAIL":
		v, ok := validation.ValidadorEmail(valor)
		if !ok {
			return domain.ErrValorInvalido
		}
		valor = v
	case "TELEFONE":
		v, ok := validation.ValidadorTelefone(valor)
		if !ok {
			return domain.ErrValorInvalido
		}
		valor = v
	case "ALEATORIA":
		// UUID já está limpo
	default:
		return domain.ErrValorInvalido
	}

	return s.repo.DeletarChavePix(ctx, cpf, tipo, valor)
}

// ============ MUDAR SENHA ============

func (s *Service) MudarSenha(ctx context.Context, input MudarSenhaInput) error {
	cpf, ok := validation.ValidadorCPF(input.CPF)
	if !ok {
		return domain.ErrCPFINvalido
	}

	conta, err := s.repo.BuscarContaPorCPF(ctx, cpf)
	if err != nil {
		return err
	}

	if !validation.VerificarSenha(input.SenhaAtual, conta.SenhaHash) {
		return domain.ErrSenhaIncorreta
	}
	if input.NovaSenha == input.SenhaAtual {
		return domain.NewErro("nova senha deve ser diferente da atual", 400)
	}
	if !validation.ValidadorSenha(input.NovaSenha) {
		return domain.ErrSenhaCurta
	}

	hash, err := validation.HashSenha(input.NovaSenha)
	if err != nil {
		return fmt.Errorf("erro ao gerar hash: %w", err)
	}

	return s.repo.AtualizarSenha(ctx, cpf, hash)
}

// ============ EDITAR DADOS ============

// EditarDados atualiza nome e/ou limite diário.
// FIX: a versão anterior passava 0 para limite quando só o nome era alterado,
// e "" para nome quando só o limite era alterado — corrompendo os dados.
// Agora busca os valores atuais e aplica apenas o que foi informado.
func (s *Service) EditarDados(ctx context.Context, input EditarDadosInput) error {
	cpf, ok := validation.ValidadorCPF(input.CPF)
	if !ok {
		return domain.ErrCPFINvalido
	}

	conta, err := s.repo.BuscarContaPorCPF(ctx, cpf)
	if err != nil {
		return err
	}

	nome := conta.Nome
	limite := conta.LimiteDiario
	alterou := false

	if n := strings.TrimSpace(input.NovoNome); len(n) >= 3 {
		nome = n
		alterou = true
	}
	if input.NovoLimite >= 100 && input.NovoLimite <= 50_000 {
		limite = int(input.NovoLimite * 100)
		alterou = true
	}

	if !alterou {
		return domain.ErrValorInvalido
	}

	return s.repo.AtualizarDados(ctx, cpf, nome, limite)
}