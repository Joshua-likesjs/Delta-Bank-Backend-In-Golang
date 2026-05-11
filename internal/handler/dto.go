package handler

type RespostaAPI struct {
    Sucesso bool        `json:"sucesso"`
    Mensagem string     `json:"mensagem"`
    Dados    interface{} `json:"dados,omitempty"`
}

type ReqCriarConta struct {
    Nome  string  `json:"nome"`
    CPF   string  `json:"cpf"`
    Senha string  `json:"senha"`
    Saldo float64 `json:"saldo_inicial"`
}

type ReqLogin struct {
    CPF   string `json:"cpf"`
    Senha string `json:"senha"`
}

type ReqPix struct {
    CPFOrigem   string  `json:"cpf_origem"`
    CPFDestino  string  `json:"cpf_destino"`
    Valor       float64 `json:"valor"`
}

type ReqDeposito struct {
    CPF   string  `json:"cpf"`
    Valor float64 `json:"valor"`
}

type ReqSaque struct {
    CPF   string  `json:"cpf"`
    Valor float64 `json:"valor"`
}

type ReqMudarSenha struct {
    CPF        string `json:"cpf"`
    SenhaAtual string `json:"senha_atual"`
    NovaSenha  string `json:"nova_senha"`
}

type ReqEditarDados struct {
    CPF        string  `json:"cpf"`
    NovoNome   string  `json:"novo_nome,omitempty"`
    NovoLimite float64 `json:"novo_limite_diario,omitempty"`
}

type ReqAdicionarChave struct {
    CPF   string `json:"cpf"`
    Tipo  string `json:"tipo"`
    Valor string `json:"valor"`
}

type ReqRemoverChave struct {
    CPF   string `json:"cpf"`
    Tipo  string `json:"tipo"`
    Valor string `json:"valor"`
}