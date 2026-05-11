package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

var db *pgx.Conn

type Sessao struct {
	CPF  string
	Nome string
}

var sessaoAtual *Sessao = nil

// ============================================================
//  FUNÇÕES DE SENHA (bcrypt)
// ============================================================

// GerarHashSenha cria um hash seguro a partir da senha em texto puro
func GerarHashSenha(senha string) (string, error) {
    // Custo padrão é 10 (pode ser aumentado para mais segurança, mas mais lento)
    bytes, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
    if err != nil {
        return "", fmt.Errorf("erro ao gerar hash da senha: %v", err)
    }
    return string(bytes), nil
}

// VerificarSenha compara a senha em texto puro com o hash armazenado
func VerificarSenha(senha string, hashArmazenado string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hashArmazenado), []byte(senha))
    return err == nil // Se err == nil, a senha está correta!
}

func main() {
	connStr := "postgres://postgres:Joelmalinda54045404@db.uvkjmwhdsxwcyifhqgpg.supabase.co:5432/postgres"

	var err error
	db, err = pgx.Connect(context.Background(), connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Erro ao conectar: %v\n", err)
		os.Exit(1)
	}
	defer db.Close(context.Background())

	for {
		if sessaoAtual == nil {
			menuInicial()
		} else {
			menuDaConta()
		}
	}
}

// ============================================================
//  BANNER E MENUS
// ============================================================



func menuInicial() {
	fmt.Println("\n┌─────────────────────────────────────────────────────┐")
	fmt.Println("                    🏦 MENU INICIAL                     ")
	fmt.Println("├─────────────────────────────────────────────────────┤")
	fmt.Println("                                                         ")
	fmt.Println("     [1] 🔐 Entrar na minha conta                        ")
	fmt.Println("     [2] ➕ Abrir nova conta                             ")
	fmt.Println("     [3] ℹ️  Sobre o Delta Bank                         ")
	fmt.Println("     [4] ❌ Sair                                         ")
	fmt.Println("                                                         ")
	fmt.Println("└─────────────────────────────────────────────────────┘")

	fmt.Print("\n   Opção: ")
	switch lerInput() {
	case "1":
		fazerLogin()
	case "2":
		criarConta()
	case "3":
		sobre()
	case "4":
		sair()
	default:
		fmt.Println("❌ Opção inválida!")
	}
}

func menuDaConta() {
	// ========== VERIFICAÇÃO DE SEGURANÇA ==========
	if sessaoAtual == nil {
		fmt.Println("\n❌ Erro: Nenhuma sessão ativa!")
		fmt.Println("   Você precisa fazer login primeiro.")
		fmt.Println("")
		return
	}
	// ============================================================

	// Tenta atualizar os dados da sessão
	atualizarSessao()

	// Verifica novamente depois de atualizar
	// (pode ter sido desativada durante o uso)
	if sessaoAtual == nil {
		fmt.Println("\n⚠️  Sessão encerrada. Redirecionando para o menu inicial...")
		return // ← VOLTA PARA O LOOP main(), não continua!
	}

	fmt.Println("\n╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("                                                                   ")
	fmt.Printf("     👤 %-53s   \n", sessaoAtual.Nome) // Agora seguro que não é nil
	fmt.Printf("     🆔 Conta: %-46s   \n", formatarCPF(sessaoAtual.CPF))
	fmt.Println("                                                                   ")
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Println("                                                                   ")
	fmt.Println("     💰 MOVIMENTAÇÕES                                            ")
	fmt.Println("     [1]  💵 Consultar Saldo                                      ")
	fmt.Println("     [2]  📋 Extrato Completo                                    ")
	fmt.Println("     [3]  💸 Fazer PIX / Transferência                         ")
	fmt.Println("     [4]  💵 Depositar Dinheiro                                 ")
	fmt.Println("     [5]  🏧 Sacar Dinheiro                                     ")
	fmt.Println("                                                                   ")
	fmt.Println("     🔑 CHAVES PIX                                               ")
	fmt.Println("     [6]  🔑 Ver Minhas Chaves                                   ")
	fmt.Println("     [7]  ➕ Adicionar Chave PIX                                ")
	fmt.Println("     [8]  🗑️  Remover Chave PIX                                  ")
	fmt.Println("                                                                   ")
	fmt.Println("     ⚙️  CONTA                                                 ")
	fmt.Println("     [9]  ✏️  Editar Meus Dados                               ")
	fmt.Println("     [10] 🔑 Mudar Senha                                       ")  // ← NOVO!
	fmt.Println("     [11] 🔒 Bloquear Conta                                     ")
	fmt.Println("     [12] 🚪 Sair da Conta (Logout)                              ")  // Número mudou!
	fmt.Println("                                                                   ")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")

	fmt.Print("\n   Opção: ")
	switch lerInput() {
case "1":
    consultarSaldo()
case "2":
    verExtrato()
case "3":
    fazerPIX()
case "4":
    depositar()
case "5":
    sacar()
case "6":
    listarChavesPix()
case "7":
    adicionarChavePixLogado()
case "8":
    removerChavePix()
case "9":
    editarDados()
case "10":
    mudarSenha()  // ← NOVO!
case "11":
    bloquearConta()
case "12":
    fmt.Printf("\n👋 Até logo, %s!\n", sessaoAtual.Nome)
    sessaoAtual = nil
default:
    fmt.Println("❌ Opção inválida!")
}
}

// ============================================================
//  1. LOGIN
// ============================================================

func fazerLogin() {
    fmt.Println("\n╔════════════════════════════════════════╗")
    fmt.Println("║     🔐 ACESSO À CONTA                   ")
    fmt.Println("╚════════════════════════════════════════╝")

    fmt.Print("\n🆔 Digite seu CPF (número da conta): ")
    cpfTxt := lerInput()

    cpfValido, ehValido := validarCPF(cpfTxt)
    if !ehValido {
        fmt.Println("❌ CPF inválido!")
        mostrarErroCPF(cpfTxt)
        return
    }

    // NOVO: Pedir senha!
    fmt.Print("🔑 Digite sua senha: ")
    senha := lerInput()
    if len(senha) < 1 {
        fmt.Println("❌ Senha não pode ser vazia!")
        return
    }

    // Buscar conta no banco (AGORA INCLUINDO A SENHA!)
    var nome string
    var saldo int
    var senhaHash string
    
    err := db.QueryRow(context.Background(), 
        "SELECT owner, balance, senha FROM contas WHERE cpf=$1", 
        cpfValido).Scan(&nome, &saldo, &senhaHash)

    if err != nil {
        fmt.Println("\n❌ Conta não encontrada!")
        fmt.Println("   Verifique se digitou o CPF corretamente.")
        fmt.Println("   Se não tem conta, escolha 'Criar nova conta'.")
        return
    }

    // NOVO: Verificar senha com bcrypt!
    if !VerificarSenha(senha, senhaHash) {
        fmt.Println("\n❌ Senha incorreta!")
        fmt.Println("   Tente novamente.")
        return
    }

    // Login bem-sucedido!
    sessaoAtual = &Sessao{
        CPF:  cpfValido,
        Nome: nome,
    }

    fmt.Println("\n✅ Login realizado com sucesso!")
    fmt.Printf("   Bem-vindo(a) de volta, %s!\n", nome)
}

// ============================================================
//  2. CRIAR CONTA
// ============================================================

func criarConta() {
    fmt.Println("\n┌─────────────────────────────────────────────────────┐")
    fmt.Println("                ➕ ABERTURA DE CONTA                   ")
    fmt.Println("└─────────────────────────────────────────────────────┘")

    fmt.Print("\n✏️  Nome completo: ")
    nome := lerInput()
    if len(nome) < 3 {
        fmt.Println("❌ Nome muito curto!")
        return
    }

    fmt.Print("🆔 Seu CPF: ")
    cpfTxt := lerInput()

    cpfValido, ok := validarCPF(cpfTxt)
    if !ok {
        fmt.Println("❌ CPF inválido!")
        return
    }

    var existente string
    err := db.QueryRow(context.Background(), "SELECT owner FROM contas WHERE cpf=$1", cpfValido).Scan(&existente)
    if err == nil {
        fmt.Printf("⚠️  CPF já cadastrado para: %s\n", existente)
        return
    }

    // ════════════════════════════════════════════════
    // NOVO: PEDIR SENHA!
    // ════════════════════════════════════════════════
    fmt.Print("🔑 Crie uma senha (mínimo 6 caracteres): ")
    senha := lerInput()
    if len(senha) < 6 {
        fmt.Println("❌ Senha muito curta! Mínimo 6 caracteres.")
        return
    }
    
    fmt.Print("🔑 Confirme sua senha: ")
    senhaConfirmacao := lerInput()
    if senha != senhaConfirmacao {
        fmt.Println("❌ As senhas não coincidem!")
        return
    }
    
    // Gerar hash da senha (NUNCA armazenar em texto puro!)
    senhaHash, err := GerarHashSenha(senha)
    if err != nil {
        fmt.Printf("❌ Erro ao processar senha: %v\n", err)
        return
    }

    fmt.Print("💰 Depósito inicial R$ (mín. R$ 10,00): ")
    saldoTxt := lerInput()
    saldo, err := strconv.ParseFloat(saldoTxt, 64)
    if err != nil || saldo < 10 {
        fmt.Println("❌ Mínimo R$ 10,00!")
        return
    }

    // NOVO: Incluir senha no INSERT!
    _, err = db.Exec(context.Background(),
        `INSERT INTO contas (cpf, owner, balance, senha) VALUES ($1, $2, $3, $4)`,
        cpfValido, nome, int(saldo*100), senhaHash)

    if err != nil {
        fmt.Printf("❌ Erro: %v\n", err)
        return
    }

    sessaoAtual = &Sessao{CPF: cpfValido, Nome: nome}

    fmt.Println("\n╔═══════════════════════════════════════════════════════════╗")
    fmt.Println("              🎉 CONTA CRIADA COM SUCESSO!                 ")
    fmt.Println("╠═══════════════════════════════════════════════════════════╣")
    fmt.Println("                                                            ")
    fmt.Println("     ┌─ DADOS DE ACESSO ────────────────────────────┐       ")
    fmt.Printf("         🏦 Conta:  %-33s        \n", formatarCPF(cpfValido))
    fmt.Println("         🔑 Senha:  A que você criou                             ")  // Mudou aqui!
    fmt.Println("     └─────────────────────────────────────────────┘       ")
    fmt.Println("                                                            ")
    fmt.Printf("     👤 Titular:  %-42s   \n", nome)
    fmt.Printf("     💰 Saldo:    R$ %-38s   \n", fmt.Sprintf("%.2f", saldo))
    fmt.Println("                                                            ")
    fmt.Println("     ✅ Você já está logado!                                ")
    fmt.Println("╚═══════════════════════════════════════════════════════════╝")

    fmt.Print("\n🔑 Cadastrar chave PIX agora? (s/n): ")
    if strings.ToLower(lerInput()) == "s" {
        adicionarChavePixLogado()
    }
}

// ============================================================
//  3. CONSULTAR SALDO
// ============================================================

func consultarSaldo() {
	var (
		saldo     int
		limite    int
		qtdChaves int
	)

	db.QueryRow(context.Background(),
		"SELECT balance, limite_diario FROM contas WHERE cpf=$1",
		sessaoAtual.CPF).Scan(&saldo, &limite)

	db.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM chaves_pix WHERE conta_cpf=$1",
		sessaoAtual.CPF).Scan(&qtdChaves)

	fmt.Println("\n╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("                      💳 RESUMO DA CONTA                     ")
	fmt.Println("╠═══════════════════════════════════════════════════════════╣")
	fmt.Println("                                                            ")
	fmt.Println("     ┌─ IDENTIFICAÇÃO ─────────────────────────────┐         ")
	fmt.Printf("         Titular:  %-35s          \n", sessaoAtual.Nome)
	fmt.Printf("         Conta:    %-35s          \n", formatarCPF(sessaoAtual.CPF))
	fmt.Println("     └─────────────────────────────────────────────┘         ")
	fmt.Println("                                                            ")
	fmt.Println("     ┌─ SALDO ────────────────────────────────────┐         ")
	fmt.Printf("         Disponível:    R$ %-24s          \n", fmt.Sprintf("%.2f", float64(saldo)/100))
	fmt.Printf("         Limite diário: R$ %-24s          \n", fmt.Sprintf("%.2f", float64(limite)/100))
	fmt.Println("     └─────────────────────────────────────────────┘         ")
	fmt.Println("                                                            ")
	fmt.Printf("     🔑 Chaves PIX: %d/5 cadastradas                          \n", qtdChaves)
	fmt.Println("                                                            ")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
}

// ============================================================
//  4. EXTRATO COMPLETO
// ============================================================

func verExtrato() {
    fmt.Println("\n╔═══════════════════════════════════════════════════════════════╗")
    fmt.Println("                                                                   ")
    fmt.Println("                       📋 EXTRATO DE MOVIMENTAÇÕES                  ")
    fmt.Println("                                                                   ")
    fmt.Printf("     🆔 Conta: %-46s   \n", formatarCPF(sessaoAtual.CPF))
    fmt.Printf("     👤 Titular: %-43s   \n", sessaoAtual.Nome)
    fmt.Println("                                                                   ")
    fmt.Println("╚═══════════════════════════════════════════════════════════════╝")

    // Buscar saldo ATUAL
    var saldoAtual int
    err := db.QueryRow(context.Background(),
        "SELECT balance FROM contas WHERE cpf=$1",
        sessaoAtual.CPF).Scan(&saldoAtual)
    
    if err != nil {
        fmt.Printf("\n❌ Erro ao buscar saldo: %v\n", err)
        return
    }

    // ════════════════════════════════════════════════════════
    // QUERY: Buscar todas as transações onde o CPF aparece
    // ════════════════════════════════════════════════════════
    rows, err := db.Query(context.Background(),
        `SELECT 
            t.tipo, 
            t.valor_centavos, 
            t.conta_cpf_remetente,
            t.conta_cpf_destinatario,
            t.created_at,
            COALESCE(c_remetente.owner, 'Desconhecido') AS nome_remetente,
            COALESCE(c_destinatario.owner, 'Desconhecido') AS nome_destinatario
         FROM transacoes t
         LEFT JOIN contas c_remetente ON t.conta_cpf_remetente = c_remetente.cpf
         LEFT JOIN contas c_destinatario ON t.conta_cpf_destinatario = c_destinatario.cpf
         WHERE 
           t.conta_cpf_remetente = $1
           OR
           t.conta_cpf_destinatario = $1
         ORDER BY t.created_at ASC`, 
        sessaoAtual.CPF)

    if err != nil {
        fmt.Printf("\n❌ Erro ao buscar extrato: %v\n", err)
        return
    }
    defer rows.Close()

    // Estrutura para armazenar transações
    type Transacao struct {
        Data      time.Time
        Tipo      string
        Valor     int
        Operacao  string  // "De: XXX" ou "Para: XXX"
        SaldoApos int
        IsEntrada bool
    }

    var transacoes []Transacao
    totalMovimentado := 0

    // ════════════════════════════════════════════════════════
    // LOOP DE LEITURA
    // ════════════════════════════════════════════════════════
    for rows.Next() {
        var tipo string
        var valor int
        var remetenteCPF, destinatarioCPF *string
        var data time.Time
        var nomeRemetente, nomeDestinatario *string

        err := rows.Scan(
            &tipo,
            &valor,
            &remetenteCPF,
            &destinatarioCPF,
            &data,
            &nomeRemetente,
            &nomeDestinatario,
        )

        if err != nil {
            fmt.Printf("\n❌ Erro ao ler transação: %v\n", err)
            continue
        }

        // ════════════════════════════════════════════════════════
        // LÓGICA PRINCIPAL: Determinar perspectiva do usuário
        // ════════════════════════════════════════════════════════
        isEntrada := false
        operacao := ""

        switch tipo {
        case "TRANSACAO_PIX":
            // Se EU sou o DESTINATÁRIO = RECEBI (entrada!)
            if destinatarioCPF != nil && *destinatarioCPF == sessaoAtual.CPF {
                isEntrada = true
                if nomeRemetente != nil && *nomeRemetente != "" {
                    operacao = fmt.Sprintf("De: %s", *nomeRemetente)
                } else {
                    operacao = "PIX Recebido"
                }
            }
            
            // Se EU sou o REMETENTE = ENVIEI (saída!)
            if remetenteCPF != nil && *remetenteCPF == sessaoAtual.CPF {
                isEntrada = false
                if nomeDestinatario != nil && *nomeDestinatario != "" {
                    operacao = fmt.Sprintf("Para: %s", *nomeDestinatario)
                } else {
                    operacao = "PIX Enviado"
                }
            }
            
        case "DEPOSITO":
            isEntrada = true
            operacao = "Depósito"
            
        case "SAQUE":
            isEntrada = false
            operacao = "Saque"
            
        default:
            operacao = tipo
        }

        // Calcular movimento total
        if isEntrada {
            totalMovimentado += valor
        } else {
            totalMovimentado -= valor
        }

        transacoes = append(transacoes, Transacao{
            Data:      data,
            Tipo:      tipo,
            Valor:     valor,
            Operacao:  operacao,
            IsEntrada: isEntrada,
        })
    }

    // Verificar erros após o loop
    if err := rows.Err(); err != nil {
        fmt.Printf("\n❌ Erro nas transações: %v\n", err)
        return
    }

    // Se não há transações
    if len(transacoes) == 0 {
        fmt.Println("\n┌─────────────────────────────────────────────────────────────┐")
        fmt.Println("                                                               ")
        fmt.Println("                    💤 NENHUMA MOVIMENTAÇÃO                      ")
        fmt.Println("                                                               ")
        fmt.Println("     Você ainda não realizou nenhuma operação.                 ")
        fmt.Println("     Suas transações aparecerão aqui.                          ")
        fmt.Println("                                                               ")
        fmt.Println("└─────────────────────────────────────────────────────────────┘")
        return
    }

    // Calcular saldo antes da primeira transação
    saldoAntesPrimeira := saldoAtual - totalMovimentado
    saldoCorrido := saldoAntesPrimeira

    // Atualizar saldos pós-transação
    for i := range transacoes {
        if transacoes[i].IsEntrada {
            saldoCorrido += transacoes[i].Valor
        } else {
            saldoCorrido -= transacoes[i].Valor
        }
        transacoes[i].SaldoApos = saldoCorrido
    }

    // Inverter para mostrar mais recentes primeiro
    for i, j := 0, len(transacoes)-1; i < j; i, j = i+1, j-1 {
        transacoes[i], transacoes[j] = transacoes[j], transacoes[i]
    }

    // ════════════════════════════════════════════════════════
    // TABELA PRINCIPAL
    // ════════════════════════════════════════════════════════
    fmt.Println("\n╔══════════════════════════════════════════════════════════════════════╗")
    fmt.Println("║  DATA/HORA           │ OPERAÇÃO               │ VALOR      │ SALDO      ║")
    fmt.Println("╠═══════════════════════╪═════════════════════════╪════════════╪════════════╣")

    totalEntradas := 0
    totalSaidas := 0

    for _, t := range transacoes {
        // Emoji baseado no tipo
        emoji := ""
        switch t.Tipo {
        case "TRANSACAO_PIX":
            if t.IsEntrada {
                emoji = "💰"  // Recebeu
            } else {
                emoji = "💸"  // Enviou
            }
        case "DEPOSITO":
            emoji = "💵"
        case "SAQUE":
            emoji = "🏧"
        default:
            emoji = "📄"
        }

        // Formatar valor com sinal
        valorFormatado := ""
        if t.IsEntrada {
            totalEntradas += t.Valor
            valorFormatado = fmt.Sprintf("+%9.2f", float64(t.Valor)/100)
        } else {
            totalSaidas += t.Valor
            valorFormatado = fmt.Sprintf("-%9.2f", float64(t.Valor)/100)
        }

        saldoFormatado := fmt.Sprintf("R$ %10.2f", float64(t.SaldoApos)/100)

        // Limitar operação para caber na coluna (20 chars)
        opDisplay := t.Operacao
        if len(opDisplay) > 20 {
            opDisplay = opDisplay[:19] + "…"
        }

        fmt.Printf("║  %s  │ %s %-19s │ %s │ %s ║\n",
            t.Data.Format("02/01/2006 15:04:05"),
            emoji,
            opDisplay,
            valorFormatado,
            saldoFormatado)
    }

    fmt.Println("╠══════════════════════════════════════════════════════════════════════╣")
    
    // Resumo final
    fmt.Println("║                         📊 RESUMO DO PERÍODO                        ║")
    fmt.Println("╠══════════════════════════════════════════════════════════════════════╣")
    fmt.Printf("║  💰 Total Recebido:    R$ %-14.2f                                ║\n", float64(totalEntradas)/100)
    fmt.Printf("║  💸 Total Enviado:     R$ %-14.2f                                ║\n", float64(totalSaidas)/100)
    fmt.Printf("║  📈 Saldo Final:       R$ %-14.2f                                ║\n", float64(saldoAtual)/100)
    fmt.Println("╚══════════════════════════════════════════════════════════════════════╝")
}

// Função auxiliar para buscar nome por CPF
func buscarNomePorCPF(cpf string) string {
	var nome string
	err := db.QueryRow(context.Background(),
		"SELECT owner FROM contas WHERE cpf=$1", cpf).Scan(&nome)
	if err != nil {
		return formatarCPF(cpf) // Retorna CPF formatado se não achar nome
	}
	return nome
}

// ============================================================
//  5. FAZER PIX
// ============================================================

func fazerPIX() {
	fmt.Println("\n┌─────────────────────────────────────────────────────┐")
	fmt.Println("                💸 TRANSFERÊNCIA PIX                    ")
	fmt.Println("└─────────────────────────────────────────────────────┘")

	var saldoAtual int
	db.QueryRow(context.Background(),
		"SELECT balance FROM contas WHERE cpf=$1",
		sessaoAtual.CPF).Scan(&saldoAtual)

	fmt.Printf("\n💰 Saldo disponível: R$ %.2f\n", float64(saldoAtual)/100)

	fmt.Print("🔑 Chave PIX ou CPF do destinatário: ")
	busca := lerInput()

	var destCPF, destNome string
	err := db.QueryRow(context.Background(),
		`SELECT c.cpf, c.owner FROM contas c
         LEFT JOIN chaves_pix k ON c.cpf = k.conta_cpf
         WHERE k.valor=$1 OR c.cpf=$2`,
		busca, removerNaoNumericos(busca)).Scan(&destCPF, &destNome)

	if err != nil {
		fmt.Println("❌ Destinatário não encontrado!")
		return
	}
	if destCPF == sessaoAtual.CPF {
		fmt.Println("❌ Não pode enviar pra si mesmo!")
		return
	}

	fmt.Print("💰 Valor R$ (ex: 50.00): ")
	valorTxt := lerInput()
	valor, err := strconv.ParseFloat(valorTxt, 64)
	if err != nil || valor <= 0 || valor < 1 {
		fmt.Println("❌ Valor inválido!")
		return
	}

	valorCent := int(valor * 100)

	if saldoAtual < valorCent {
		fmt.Println("❌ Saldo insuficiente!")
		return
	}

	// Verificar limite diário usado
	var usadoHoje int
	db.QueryRow(context.Background(),
		`SELECT COALESCE(SUM(valor_centavos), 0) FROM transacoes 
         WHERE conta_cpf_origem=$1 AND tipo='PIX_ENVIADO' 
         AND DATE(created_at)=CURRENT_DATE`,
		sessaoAtual.CPF).Scan(&usadoHoje)

	var limiteDiario int
	db.QueryRow(context.Background(),
		"SELECT limite_diario FROM contas WHERE cpf=$1",
		sessaoAtual.CPF).Scan(&limiteDiario)

	if (usadoHoje + valorCent) > limiteDiario {
		fmt.Printf("❌ Limite diário excedido! Usado: R$ %.2f de R$ %.2f\n",
			float64(usadoHoje)/100, float64(limiteDiario)/100)
		return
	}

	fmt.Printf("\n Confirme transferência de R$ %.2f para %s? (s/n): ", valor, destNome)
	if strings.ToLower(lerInput()) != "s" {
		fmt.Println("Cancelado.")
		return
	}

	tx, err := db.Begin(context.Background())
	if err != nil {
		fmt.Printf("❌ ERRO CRÍTICO ao iniciar transação: %v\n", err)
		return
	}

	fmt.Println("\n🔄 Executando transação bancária...")

	// Débito na conta de origem
	_, err = tx.Exec(context.Background(),
		"UPDATE contas SET balance=balance-$1, updated_at=NOW() WHERE cpf=$2",
		valorCent, sessaoAtual.CPF)
	if err != nil {
		tx.Rollback(context.Background())
		fmt.Printf("❌ Erro no débito: %v\n", err)
		return
	}
	fmt.Println("   ✓ Débito realizado")

	// Crédito na conta de destino
	_, err = tx.Exec(context.Background(),
		"UPDATE contas SET balance=balance+$1, updated_at=NOW() WHERE cpf=$2",
		valorCent, destCPF)
	if err != nil {
		tx.Rollback(context.Background())
		fmt.Printf("❌ Erro no crédito: %v\n", err)
		return
	}
	fmt.Println("   ✓ Crédito realizado")

	// Registrar transação
	_, err = tx.Exec(context.Background(),
    `INSERT INTO transacoes (conta_cpf_remetente, conta_cpf_destinatario, tipo, valor_centavos)
     VALUES ($1, $2, 'TRANSACAO_PIX', $3)`,
    sessaoAtual.CPF, destCPF, valorCent)
	if err != nil {
		tx.Rollback(context.Background())
		fmt.Printf("❌ Erro ao registrar transação: %v\n", err)
		return
	}
	fmt.Println("   ✓ Transação registrada")

	// COMMIT FINAL
	err = tx.Commit(context.Background())
	if err != nil {
		fmt.Printf("❌ ERRO NO COMMIT: %v\n", err)
		return
	}

	fmt.Println("   ✅ Transação confirmada no banco!")
	imprimirComprovante("PIX ENVIADO", destNome, destCPF, valor)
}

// ============================================================
//  6. DEPOSITAR
// ============================================================

func depositar() {
	fmt.Println("\n┌─────────────────────────────────────────────────────┐")
	fmt.Println("                 💵 DEPÓSITO EM DINHEIRO                ")
	fmt.Println("└─────────────────────────────────────────────────────┘")

	var saldo int
	db.QueryRow(context.Background(),
		"SELECT balance FROM contas WHERE cpf=$1",
		sessaoAtual.CPF).Scan(&saldo)

	fmt.Printf("\n💰 Saldo atual: R$ %.2f\n", float64(saldo)/100)
	fmt.Print("💵 Valor do depósito R$ (máx. R$ 10.000): ")

	valorTxt := lerInput()
	valor, err := strconv.ParseFloat(valorTxt, 64)
	if err != nil || valor <= 0 || valor > 10000 {
		fmt.Println("❌ Valor inválido (máx. R$ 10.000)!")
		return
	}

	valorCent := int(valor * 100)

	fmt.Printf("\n Confirmar depósito de R$ %.2f? (s/n): ", valor)
	if strings.ToLower(lerInput()) != "s" {
		return
	}

	tx, _ := db.Begin(context.Background())

	tx.Exec(context.Background(),
		"UPDATE contas SET balance=balance+$1, updated_at=NOW() WHERE cpf=$2",
		valorCent, sessaoAtual.CPF)

	tx.Exec(context.Background(),
    `INSERT INTO transacoes (conta_cpf_remetente, conta_cpf_destinatario, tipo, valor_centavos)
     VALUES ($1, $1, 'DEPOSITO', $2)`,
    sessaoAtual.CPF, valorCent)

	tx.Commit(context.Background())

	var novoSaldo int
	db.QueryRow(context.Background(),
		"SELECT balance FROM contas WHERE cpf=$1",
		sessaoAtual.CPF).Scan(&novoSaldo)

	fmt.Println("\n✅ Depósito realizado!")
	fmt.Println("╔═════════════════════════════════╗")
	fmt.Printf("    Depositado: R$ %-15s   \n", fmt.Sprintf("%.2f", valor))
	fmt.Printf("    Novo saldo: R$ %-16s   \n", fmt.Sprintf("%.2f", float64(novoSaldo)/100))
	fmt.Println("╚═════════════════════════════════╝")
}

// ============================================================
//  7. SACAR
// ============================================================

func sacar() {
	fmt.Println("\n┌─────────────────────────────────────────────────────┐")
	fmt.Println("                  🏧 SAQUE EM DINHEIRO                  ")
	fmt.Println("└─────────────────────────────────────────────────────┘")

	var saldo int
	db.QueryRow(context.Background(),
		"SELECT balance FROM contas WHERE cpf=$1",
		sessaoAtual.CPF).Scan(&saldo)

	fmt.Printf("\n💰 Saldo disponível: R$ %.2f", float64(saldo)/100)
	fmt.Print("🏧 Valor do saque R$ (múltiplo de 10, máx. R$ 2.000): ")

	valorTxt := lerInput()
	valor, err := strconv.ParseFloat(valorTxt, 64)
	if err != nil || valor <= 0 {
		fmt.Println("❌ Inválido!")
		return
	}

	if int(valor*100)%1000 != 0 {
		fmt.Println("❌ Deve ser múltiplo de R$ 10,00!")
		return
	}
	if valor > 2000 {
		fmt.Println("❌ Máximo R$ 2.000,00 por saque!")
		return
	}

	valorCent := int(valor * 100)
	if saldo < valorCent {
		fmt.Println("❌ Saldo insuficiente!")
		return
	}

	fmt.Printf("\n Confirmar saque de R$ %.2f? (s/n): ", valor)
	if strings.ToLower(lerInput()) != "s" {
		return
	}

	tx, _ := db.Begin(context.Background())

	tx.Exec(context.Background(),
		"UPDATE contas SET balance=balance-$1, updated_at=NOW() WHERE cpf=$2",
		valorCent, sessaoAtual.CPF)

	tx.Exec(context.Background(),
    `INSERT INTO transacoes (conta_cpf_remetente, conta_cpf_destinatario, tipo, valor_centavos)
     VALUES ($1, $1, 'SAQUE', $2)`,
    sessaoAtual.CPF, valorCent)

	tx.Commit(context.Background())

	var novoSaldo int
	db.QueryRow(context.Background(),
		"SELECT balance FROM contas WHERE cpf=$1",
		sessaoAtual.CPF).Scan(&novoSaldo)

	fmt.Println("\n✅ Saque realizado!")
	fmt.Println("╔═════════════════════════════════╗")
	fmt.Printf("    Sacado:  R$ %-17s   \n", fmt.Sprintf("%.2f", valor))
	fmt.Printf("    Saldo:   R$ %-17s   \n", fmt.Sprintf("%.2f", float64(novoSaldo)/100))
	fmt.Println("╚═════════════════════════════════╝")
}

// ============================================================
//  8. CHAVES PIX - LISTAR
// ============================================================

func listarChavesPix() {
	rows, err := db.Query(context.Background(),
		`SELECT tipo, valor, created_at FROM chaves_pix 
         WHERE conta_cpf=$1 ORDER BY created_at`, sessaoAtual.CPF)

	if err != nil {
		fmt.Printf("❌ Erro: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Println("\n┌────────────────────────────────────────────────────┐")
	fmt.Println("              📋 SUAS CHAVES PIX CADASTRADAS            ")
	fmt.Println("├────────┬────────────────────────────────────────────┤")
	fmt.Println("    TIPO      VALOR DA CHAVE                               ")
	fmt.Println("├────────┼────────────────────────────────────────────┤")

	qtd := 0
	for rows.Next() {
		var t, v string
		var d time.Time
		rows.Scan(&t, &v, &d)

		e := map[string]string{"CPF": "🆔", "Email": "📧", "Telefone": "📱", "Aleatoria": "🎲"}[t]
		fmt.Printf("   %s %-6s   %-43s   \n", e, t, v)
		qtd++
	}

	if qtd == 0 {
		fmt.Println("                                                          ")
		fmt.Println("    Nenhuma chave cadastrada yet!                          ")
	} else {
		fmt.Println("├────────┴────────────────────────────────────────────┤")
		fmt.Printf("    Total: %d/5 chaves utilizadas                           \n", qtd)
	}
	fmt.Println("└────────────────────────────────────────────────────┘")
}

// ============================================================
//  9. CHAVES PIX - ADICIONAR
// ============================================================

func adicionarChavePix() {
	if sessaoAtual != nil {
		adicionarChavePixLogado()
	}
}

func adicionarChavePixLogado() {
	var qtd int
	db.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM chaves_pix WHERE conta_cpf=$1",
		sessaoAtual.CPF).Scan(&qtd)

	if qtd >= 5 {
		fmt.Printf("❌ Limite de 5 chaves atingido!\n")
		listarChavesPix()
		return
	}

	fmt.Printf("\n📝 Chaves usadas: %d/5\n", qtd)
	fmt.Println("\nTipo de chave:")
	fmt.Println("  [1] 🆔 CPF")
	fmt.Println("  [2] 📧 E-mail")
	fmt.Println("  [3] 📱 Telefone")
	fmt.Println("  [4] 🎲 Aleatória")
	fmt.Print("Opção: ")

	tipoOp := lerInput()
	var tipo, valor string

	switch tipoOp {
	case "1":
		tipo = "CPF"
		valor = sessaoAtual.CPF
		fmt.Printf("Usando CPF: %s\n", formatarCPF(valor))
	case "2":
		tipo = "Email"
		fmt.Print("E-mail: ")
		e := lerInput()
		var ok bool
		valor, ok = validarEmail(e)
		if !ok {
			return
		}
	case "3":
		tipo = "Telefone"
		fmt.Print("Telefone: ")
		f := lerInput()
		var ok bool
		valor, ok = validarTelefone(f)
		if !ok {
			return
		}
		valor = formatarTelefone(valor)
	case "4":
		tipo = "Aleatoria"
		valor = gerarChaveAleatoria()
		fmt.Printf("Chave: %s\n", valor)
	default:
		fmt.Println("❌ Inválido!")
		return
	}

	// Verificar se já existe no sistema
	var dono string
	err := db.QueryRow(context.Background(),
		`SELECT c.owner FROM contas c JOIN chaves_pix k ON c.cpf=k.conta_cpf WHERE k.valor=$1`,
		valor).Scan(&dono)
	if err == nil {
		fmt.Printf("❌ Já pertence a: %s\n", dono)
		return
	}

	// Verificar se já tem deste tipo
	var existente string
	err = db.QueryRow(context.Background(),
		"SELECT valor FROM chaves_pix WHERE conta_cpf=$1 AND tipo=$2",
		sessaoAtual.CPF, tipo).Scan(&existente)
	if err == nil {
		fmt.Printf("⚠️  Já tem chave '%s': %s\nSubstituir? (s/n): ", tipo, existente)
		if strings.ToLower(lerInput()) == "s" {
			db.Exec(context.Background(), "DELETE FROM chaves_pix WHERE conta_cpf=$1 AND tipo=$2", sessaoAtual.CPF, tipo)
		} else {
			return
		}
	}

	_, err = db.Exec(context.Background(),
		`INSERT INTO chaves_pix (conta_cpf, tipo, valor, created_at) VALUES ($1, $2, $3, NOW())`,
		sessaoAtual.CPF, tipo, valor)

	if err != nil {
		fmt.Printf("❌ Erro: %v\n", err)
		return
	}

	fmt.Println("\n✅ Chave cadastrada!")
	fmt.Println("╔═════════════════════════════════╗")
	fmt.Printf("    Tipo:  %-24s   \n", tipo)
	fmt.Printf("    Chave: %-24s   \n", valor)
	fmt.Println("╚═════════════════════════════════╝")
}

// ============================================================
//  10. CHAVES PIX - REMOVER
// ============================================================

func removerChavePix() {
	listarChavesPix()

	var qtd int
	db.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM chaves_pix WHERE conta_cpf=$1",
		sessaoAtual.CPF).Scan(&qtd)

	if qtd == 0 {
		fmt.Println("Nenhuma chave para remover!")
		return
	}

	fmt.Print("\nDigite o TIPO da chave a remover (CPF/Email/Telefone/Aleatoria): ")
	tipo := lerInput()

	fmt.Print("Digite o VALOR da chave: ")
	valor := lerInput()

	result, err := db.Exec(context.Background(),
		"DELETE FROM chaves_pix WHERE conta_cpf=$1 AND tipo=$2 AND valor=$3",
		sessaoAtual.CPF, tipo, valor)

	if err != nil {
		fmt.Printf("❌ Erro: %v\n", err)
		return
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		fmt.Println("❌ Chave não encontrada!")
		return
	}

	fmt.Printf("\n✅ Chave removida! (%d linha afetada)\n", rowsAffected)
}

// ============================================================
//  11. EDITAR DADOS
// ============================================================

func editarDados() {
	fmt.Println("\n┌─────────────────────────────────────────────────────┐")
	fmt.Println("              ✏️  EDITAR MEUS DADOS                     ")
	fmt.Println("└─────────────────────────────────────────────────────┘")

	fmt.Println("\nDados atuais:")
	fmt.Printf("  Nome: %s\n", sessaoAtual.Nome)
	fmt.Printf("  CPF:  %s\n", formatarCPF(sessaoAtual.CPF))

	fmt.Println("\nO que deseja alterar?")
	fmt.Println("  [1] Nome")
	fmt.Println("  [2] Limite diário de transferência")
	fmt.Println("  [3] Voltar")
	fmt.Print("Opção: ")

	switch lerInput() {
	case "1":
		fmt.Print("\nNovo nome: ")
		novoNome := lerInput()
		if len(novoNome) < 3 {
			fmt.Println("❌ Muito curto!")
			return
		}

		db.Exec(context.Background(),
			"UPDATE contas SET owner=$1, updated_at=NOW() WHERE cpf=$2",
			novoNome, sessaoAtual.CPF)

		sessaoAtual.Nome = novoNome
		fmt.Printf("✅ Nome alterado para: %s\n", novoNome)

	case "2":
		var limiteAtual int
		db.QueryRow(context.Background(),
			"SELECT limite_diario FROM contas WHERE cpf=$1",
			sessaoAtual.CPF).Scan(&limiteAtual)

		fmt.Printf("\nLimite diário atual: R$ %.2f\n", float64(limiteAtual)/100)
		fmt.Print("Novo limite diário R$ (mín. 100, máx. 50.000): ")

		novoLimiteTxt := lerInput()
		novoLimite, err := strconv.ParseFloat(novoLimiteTxt, 64)
		if err != nil || novoLimite < 100 || novoLimite > 50000 {
			fmt.Println("❌ Valor entre R$ 100 e R$ 50.000!")
			return
		}

		db.Exec(context.Background(),
			"UPDATE contas SET limite_diario=$1, updated_at=NOW() WHERE cpf=$2",
			int(novoLimite*100), sessaoAtual.CPF)

		fmt.Printf("✅ Limite alterado para: R$ %.2f\n", novoLimite)

	case "3":
		return
	default:
		fmt.Println("❌ Opção inválida!")
	}
}

// ============================================================
//  13. MUDAR SENHA
// ============================================================

func mudarSenha() {
    fmt.Println("\n┌─────────────────────────────────────────────────────┐")
    fmt.Println("              🔑 ALTERAÇÃO DE SENHA                     ")
    fmt.Println("└─────────────────────────────────────────────────────┘")

    fmt.Printf("\n🆔 Conta: %s\n", formatarCPF(sessaoAtual.CPF))
    fmt.Printf("👤 Titular: %s\n", sessaoAtual.Nome)

    // Passo 1: Pedir senha ATUAL para confirmar identidade
    fmt.Print("\n🔑 Digite sua senha ATUAL: ")
    senhaAtual := lerInput()

    // Buscar hash atual do banco
    var senhaHash string
    err := db.QueryRow(context.Background(),
        "SELECT senha FROM contas WHERE cpf=$1",
        sessaoAtual.CPF).Scan(&senhaHash)

    if err != nil {
        fmt.Println("\n❌ Erro ao buscar dados da conta!")
        return
    }

    // Verificar se a senha atual está correta
    if !VerificarSenha(senhaAtual, senhaHash) {
        fmt.Println("\n❌ Senha ATUAL incorreta!")
        fmt.Println("   Não foi possível alterar a senha.")
        return
    }

    // Passo 2: Pedir nova senha
    fmt.Print("\n✏️  Digite a NOVA senha (mínimo 6 caracteres): ")
    novaSenha := lerInput()

    if len(novaSenha) < 6 {
        fmt.Println("❌ Nova senha muito curta! Mínimo 6 caracteres.")
        return
    }

    // Verificar se a nova senha é diferente da atual
    if novaSenha == senhaAtual {
        fmt.Println("⚠️  A nova senha deve ser diferente da atual!")
        return
    }

    // Passo 3: Confirmar nova senha
    fmt.Print("🔑 Confirme a NOVA senha: ")
    novaSenhaConfirmacao := lerInput()

    if novaSenha != novaSenhaConfirmacao {
        fmt.Println("❌ As senhas não coincidem!")
        return
    }

    // Passo 4: Gerar hash da nova senha
    novoHash, err := GerarHashSenha(novaSenha)
    if err != nil {
        fmt.Printf("❌ Erro ao processar nova senha: %v\n", err)
        return
    }

    // Passo 5: Atualizar no banco
    _, err = db.Exec(context.Background(),
        "UPDATE contas SET senha=$1, updated_at=NOW() WHERE cpf=$2",
        novoHash, sessaoAtual.CPF)

    if err != nil {
        fmt.Printf("❌ Erro ao atualizar senha: %v\n", err)
        return
    }

    // Sucesso!
    fmt.Println("\n╔═════════════════════════════════╗")
    fmt.Println("   ✅ SENHA ALTERADA COM SUCESSO!   ")
    fmt.Println("╚═════════════════════════════════╝")
    fmt.Println("")
    fmt.Println("   Use sua nova senha no próximo login.")
    fmt.Println("")
}

// ============================================================
//  12. BLOQUEAR CONTA
// ============================================================

func bloquearConta() {
	fmt.Println("\n┌─────────────────────────────────────────────────────┐")
	fmt.Println("              ⚠️  BLOQUEIO DE CONTA                      ")
	fmt.Println("└─────────────────────────────────────────────────────┘")

	fmt.Println("\n⚠️  ATENÇÃO: Ao bloquear sua conta:")
	fmt.Println("   • Você não poderá fazer operações")
	fmt.Println("   • Suas chaves PIX serão desativadas")
	fmt.Println("   • Para reativar, entre em contato com o suporte")
	fmt.Println("")
	fmt.Printf("Confirma bloqueio da conta %s? (digite 'CONFIRMAR'): ", formatarCPF(sessaoAtual.CPF))

	if lerInput() != "CONFIRMAR" {
		fmt.Println("Operação cancelada.")
		return
	}

	tx, _ := db.Begin(context.Background())

	tx.Exec(context.Background(),
		"UPDATE contas SET ativa=false, updated_at=NOW() WHERE cpf=$1",
		sessaoAtual.CPF)

	// Remover todas as chaves PIX
	tx.Exec(context.Background(),
		"DELETE FROM chaves_pix WHERE conta_cpf=$1", sessaoAtual.CPF)

	tx.Commit(context.Background())

	fmt.Println("\n✅ Conta bloqueada com sucesso!")
	fmt.Printf("   CPF: %s\n", formatarCPF(sessaoAtual.CPF))
	fmt.Println("   Para reativar, contate: suporte@deltabank.com")

	sessaoAtual = nil
}

// ============================================================
//  FUNÇÕES AUXILIARES
// ============================================================

func atualizarSessao() {
	// Se não tem ninguém logado, nem tenta atualizar
	if sessaoAtual == nil {
		return
	}

	var nome string
	err := db.QueryRow(context.Background(),
		"SELECT owner FROM contas WHERE cpf=$1",
		sessaoAtual.CPF).Scan(&nome)

	// Em vez de crashar, apenas faz logout silencioso
	if err != nil {
		// Log do erro para debug (remova em produção)
		fmt.Printf("\n[DEBUG] Erro ao buscar sessão: %v\n", err)
		sessaoAtual = nil
		return

	}

	// Só atualiza se conseguiu buscar o nome
	sessaoAtual.Nome = nome
}

func imprimirComprovante(tipo, destNome, destCPF string, valor float64) {
	fmt.Println("\n╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("                                                             ")
	fmt.Println("                ✅ OPERAÇÃO REALIZADA COM SUCESSO               ")
	fmt.Println("                                                             ")
	fmt.Println("                      📄 COMPROVANTE                         ")
	fmt.Println("╠═══════════════════════════════════════════════════════════════╣")
	fmt.Println("                                                             ")
	fmt.Println("     DELTA BANK                                               ")
	fmt.Printf("     Data/Hora: %-47s   \n", time.Now().Format("02/01/2006 15:04:05"))
	fmt.Println("                                                             ")
	fmt.Println("     ┌─ PAGADOR ────────────────────────────────────┐       ")
	fmt.Printf("         CPF:  %-38s        \n", formatarCPF(sessaoAtual.CPF))
	fmt.Printf("         Nome: %-37s        \n", sessaoAtual.Nome)
	fmt.Println("     └─────────────────────────────────────────────┘       ")
	fmt.Println("                                                             ")
	fmt.Println("     ┌─ BENEFICIÁRIO ─────────────────────────────┐       ")
	fmt.Printf("         CPF:  %-38s        \n", formatarCPF(destCPF))
	fmt.Printf("         Nome: %-37s        \n", destNome)
	fmt.Println("     └─────────────────────────────────────────────┘       ")
	fmt.Println("                                                             ")
	fmt.Println("     ┌─ VALOR ─────────────────────────────────────┐       ")
	fmt.Printf("         R$ %-41s        \n", fmt.Sprintf("%.2f", valor))
	fmt.Println("     └─────────────────────────────────────────────┘       ")
	fmt.Println("                                                             ")
	fmt.Println("     Status: ✅ PAGO                                       ")
	fmt.Println("     Tipo: ", tipo)
	fmt.Println("                                                             ")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
}

func sobre() {
	fmt.Println("\n╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("                                                               ")
	fmt.Println("                      SOBRE O DELTA BANK                       ")
	fmt.Println("                                                               ")
	fmt.Println("╠═══════════════════════════════════════════════════════════╣")
	fmt.Println("                                                               ")
	fmt.Println("     Versão:     7.0 Completa                                   ")
	fmt.Println("     Desenvolvedor Backend: Josué Gabriel Freitas da Silva              ")
	fmt.Println("     Desenvolvedor Frontend: Nelson Alexandre Breves Chíxaro Golveia              ")
	fmt.Println("     Desenvolvedor Frontend: Edward              ")
	fmt.Println("     Tecnologia:  Go (Golang) + PostgreSQL/Supabase            ")
	fmt.Println("     Ano:         2025                                           ")
	fmt.Println("                                                               ")
	fmt.Println("     Um sistema bancário educacional, desenvolvido como         ")
	fmt.Println("     projeto pessoal de aprendizado em programação.             ")
	fmt.Println("                                                               ")
	fmt.Println("     © 2025 Delta Bank. Todos os direitos reservados.           ")
	fmt.Println("                                                               ")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
}

func sair() {
	fmt.Println("\n👋 Obrigado por usar o Delta Bank!")
	fmt.Println("   Tenha um excelente dia!")
	os.Exit(0)
}

// ============================================================
//  VALIDAÇÕES
// ============================================================

func validarCPF(cpf string) (string, bool) {
	c := removerNaoNumericos(cpf)
	if len(c) != 11 || todosDigitosIguais(c) {
		return "", false
	}
	d1 := calcDigito(c[:9], 10)
	if d1 != int(c[9]-'0') {
		return "", false
	}
	d2 := calcDigito(c[:10], 11)
	if d2 != int(c[10]-'0') {
		return "", false
	}
	return c, true
}

func calcDigito(d string, p int) int {
	s := 0
	w := p
	for i := 0; i < len(d); i++ {
		s += int(d[i]-'0') * w
		w--
	}
	r := s % 11
	if r < 2 {
		return 0
	}
	return 11 - r
}

func todosDigitosIguais(s string) bool {
	for i := 1; i < len(s); i++ {
		if s[i] != s[0] {
			return false
		}
	}
	return true
}

func formatarCPF(c string) string {
	if len(c) == 11 {
		return c[:3] + "." + c[3:6] + "." + c[6:9] + "-" + c[9:]
	}
	return c
}

func validarEmail(e string) (string, bool) {
	e = strings.TrimSpace(strings.ToLower(e))
	if !regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`).MatchString(e) {
		return "", false
	}
	p := strings.Split(e, "@")
	if len(p) != 2 || len(p[1]) < 5 || !strings.Contains(p[1], ".") {
		return "", false
	}
	return e, true
}

func validarTelefone(t string) (string, bool) {
	n := removerNaoNumericos(t)
	if len(n) > 11 && n[:2] == "55" {
		n = n[2:]
	}
	if len(n) != 10 && len(n) != 11 {
		return "", false
	}
	if !validarDDD(n[:2]) || n[2] == '0' || (len(n) == 11 && n[2] != '9') {
		return "", false
	}
	return n, true
}

func validarDDD(d string) bool {
	ddds := map[string]bool{
		"11": true, "12": true, "13": true, "14": true, "15": true, "16": true, "17": true, "18": true, "19": true,
		"21": true, "22": true, "24": true, "27": true, "28": true, "31": true, "32": true, "33": true, "34": true,
		"35": true, "37": true, "38": true, "41": true, "42": true, "43": true, "44": true, "45": true, "46": true,
		"47": true, "48": true, "49": true, "51": true, "53": true, "54": true, "55": true, "61": true, "62": true,
		"63": true, "64": true, "65": true, "66": true, "67": true, "71": true, "73": true, "74": true, "75": true,
		"77": true, "78": true, "79": true, "81": true, "82": true, "83": true, "84": true, "85": true, "86": true,
		"87": true, "88": true, "89": true, "91": true, "92": true, "93": true, "94": true, "95": true, "96": true,
		"97": true, "98": true, "99": true,
	}
	return ddds[d]
}

func formatarTelefone(t string) string {
	if len(t) == 11 {
		return "(" + t[:2] + ") " + t[2:7] + "-" + t[7:]
	}
	if len(t) == 10 {
		return "(" + t[:2] + ") " + t[2:6] + "-" + t[6:]
	}
	return t
}

func gerarChaveAleatoria() string {
	u := make([]byte, 16)
	s := time.Now().UnixNano()
	for i := 0; i < 16; i++ {
		s = (s*1103515245 + 12345) & 0x7fffffff
		u[i] = byte(s % 100)
	}
	u[6] = (u[6] & 0x0f) | 0x40
	u[8] = (u[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}

func removerNaoNumericos(s string) string {
	return regexp.MustCompile(`[^0-9]`).ReplaceAllString(s, "")
}

func mostrarErroCPF(cpf string) {
	c := removerNaoNumericos(cpf)
	if len(c) != 11 {
		fmt.Printf(" Tem %d dígitos (precisa 11)\n", len(c))
	} else if todosDigitosIguais(c) {
		fmt.Println(" Dígitos iguais")
	} else {
		fmt.Println(" Dígitos verificadores errados")
	}
}

func lerInput() string {
	r := bufio.NewReader(os.Stdin)
	t, _ := r.ReadString('\n')
	return strings.TrimSpace(t)
}
