package domain

import (
    "fmt"
    "time"
)

// ============================================================
//  ENTIDADES DO DOMÍNIO
// ============================================================

// Conta representa uma conta bancária
type Conta struct {
    CPF           string    `json:"cpf"`
    Nome          string    `json:"nome"`
    SaldoCentavos int       `json:"saldo_centavos"`
    SenhaHash     string    `json:"-"` // Nunca serializar no JSON!
    LimiteDiario  int       `json:"limite_diario"`
    CriadaEm      time.Time `json:"criada_em"`
    AtualizadaEm  time.Time `json:"atualizada_em"`
}

// SaldoFormatado retorna o saldo formatado em Reais
func (c *Conta) SaldoFormatado() string {
    return fmt.Sprintf("%.2f", float64(c.SaldoCentavos)/100)
}

// LimiteDiarioFormatado retorna o limite formatado em Reais
func (c *Conta) LimiteDiarioFormatado() string {
    return fmt.Sprintf("%.2f", float64(c.LimiteDiario)/100)
}

// Transacao representa uma movimentação financeira
type Transacao struct {
    ID              int        `json:"id"`
    Tipo            string     `json:"tipo"` // TRANSACAO_PIX, DEPOSITO, SAQUE
    ValorCentavos   int        `json:"valor_centavos"`
    CPFRemetente    string     `json:"cpf_remetente"`
    CPFDestinatario  string     `json:"cpf_destinatario"`
    NomeRemetente   string     `json:"nome_remetente,omitempty"`
    NomeDestinatario string     `json:"nome_destinatario,omitempty"`
    DataHora        time.Time  `json:"data_hora"`
}

// ValorFormatado retorna o valor formatado com sinal
func (t *Transacao) ValorFormatado() string {
    valor := float64(t.ValorCentavos) / 100
    if t.EhEntradaParaCPF("") { // método auxiliar
        return fmt.Sprintf("+%.2f", valor)
    }
    return fmt.Sprintf("-%.2f", valor)
}

// EhEntradaParaCPF verifica se a transação foi entrada para o CPF informado
func (t *Transacao) EhEntradaParaCPF(cpfUsuario string) bool {
    switch t.Tipo {
    case "TRANSACAO_PIX":
        return t.CPFDestinatario == cpfUsuario
    case "DEPOSITO":
        return true
    default:
        return false
    }
}

// OperacaoDescricao retorna descrição amigável da operação
func (t *Transacao) OperacaoDescricao(cpfUsuario string) string {
    switch t.Tipo {
    case "TRANSACAO_PIX":
        if t.CPFDestinatario == cpfUsuario {
            return fmt.Sprintf("De: %s", t.NomeRemetente)
        }
        return fmt.Sprintf("Para: %s", t.NomeDestinatario)
    case "DEPOSITO":
        return "Depósito"
    case "SAQUE":
        return "Saque"
    default:
        return t.Tipo
    }
}

// ChavePix representa uma chave PIX cadastrada
type ChavePix struct {
    ID        int        `json:"id"`
    Tipo      string     `json:"tipo"` // CPF, Email, Telefone, Aleatoria
    Valor     string     `json:"valor"`
    CPFConta  string     `json:"cpf_conta"`
    CriadaEm  time.Time  `json:"criada_em"`
}