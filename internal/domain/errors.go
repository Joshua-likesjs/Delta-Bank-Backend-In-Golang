package domain

// ============================================================
//  ERROS DO DOMÍNIO (Business Errors)
// ============================================================

// ErroDominio representa um erro de negócio/validação
type ErroDominio struct {
    Mensagem string `json:"mensagem"`
    Codigo    int    `json:"codigo,omitempty"` // Código HTTP opcional
}

// Error implementa a interface error
func (e *ErroDominio) Error() string {
    return e.Mensagem
}

// Erros prontos para usar no código
var (
    // Conta
    ErrContaNaoEncontrada        = NewErro("conta não encontrada", 404)
    ErrCPFJaCadastrado           = NewErro("CPF já cadastrado", 409)
    ErrCPFINvalido               = NewErro("CPF inválido", 400)
    ErrNomeCurto                 = NewErro("nome deve ter no mínimo 3 caracteres", 400)
    ErrSenhaCurta                = NewErro("senha deve ter no mínimo 6 caracteres", 400)
    ErrSenhaIncorreta            = NewErro("senha incorreta", 401)
    ErrContaBloqueada            = NewErro("conta bloqueada por muitas tentativas. Tente mais tarde.", 423)
    
    // Operações financeiras
    ErrSaldoInsuficiente         = NewErro("saldo insuficiente", 400)
    ErrLimiteExcedido            = NewErro("limite diário excedido", 400)
    ErrValorInvalido             = NewErro("valor inválido", 400)
    ErrDestinatarioNaoEncontrado = NewErro("destinatário não encontrado", 404)
    ErrAutoTransferencia         = NewErro("não pode transferir para si mesmo", 400)
    ErrExtrato                   = NewErro("não foi possível acessar o extrato", 400)
    
    
    // PIX / Chaves
    ErrChaveJaExistente          = NewErro("chave PIX já cadastrada no sistema", 409)
    ErrLimiteChavesAtingido      = NewErro("limite de 5 chaves PIX atingido", 400)
    ErrChaveNaoEncontrada        = NewErro("chave PIX não encontrada", 404)
    
    // Genéricos
    ErrInterno                   = NewErro("erro interno do servidor", 500)
    ErrNaoAutorizado             = NewErro("não autorizado", 403)
)

// NewErro cria um novo erro de domínio
func NewErro(mensagem string, codigo int) *ErroDominio {
    return &ErroDominio{
        Mensagem: mensagem,
        Codigo:    codigo,
    }
}

// EhDominio verifica se um erro é do tipo ErroDominio
func EhDominio(err error) bool {
    _, ok := err.(*ErroDominio)
    return ok
}