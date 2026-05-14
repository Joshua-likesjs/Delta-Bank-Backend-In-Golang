
# Delta Bank Backend

API RESTful para um sistema bancário digital construída em **Golang** com **Clean Architecture** e **PostgreSQL**. O projeto implementa operações bancárias essenciais como criação de contas, autenticação, transferências via PIX, depósitos, saques e gerenciamento de chaves PIX — com validações robustas de regras de negócio, segurança com hash de senhas via bcrypt, pool de conexões, logging estruturado e middleware de produção.

> **API em produção:** [https://delta-bank-backend-in-golang-production.up.railway.app](https://delta-bank-backend-in-golang-production.up.railway.app)

## Sumário

- [Deploy](#deploy)
- [Tecnologias](#tecnologias)
- [Arquitetura](#arquitetura)
- [Middleware](#middleware)
- [Pré-requisitos](#pré-requisitos)
- [Instalação e Execução](#instalação-e-execução)
- [Variáveis de Ambiente](#variáveis-de-ambiente)
- [Estrutura do Banco de Dados](#estrutura-do-banco-de-dados)
- [Endpoints da API](#endpoints-da-api)
- [Exemplos de Requisição](#exemplos-de-requisição)
- [Validações e Regras de Negócio](#validações-e-regras-de-negócio)
- [Tratamento de Erros](#tratamento-de-erros)
- [Estrutura de Diretórios](#estrutura-de-diretórios)

## Deploy

A API está hospedada em produção no **Railway** e pode ser acessada diretamente:

```
https://delta-bank-backend-in-golang-production.up.railway.app
```

Todas as rotas estão disponíveis sob o prefixo `/api`. Exemplo rápido:

```bash
# Criar uma conta na API de produção
curl -X POST https://delta-bank-backend-in-golang-production.up.railway.app/api/contas \
  -H "Content-Type: application/json" \
  -d '{"nome": "João Silva", "cpf": "12345678909", "senha": "minhasenha", "saldo_inicial": 100.00}'

# Consultar saldo
curl https://delta-bank-backend-in-golang-production.up.railway.app/api/saldo/12345678909
```

O deploy é feito automaticamente via Railway a partir do repositório GitHub, com as variáveis de ambiente (`DATABASE_URL` e `PORT`) configuradas no painel da plataforma.

## Tecnologias

| Tecnologia | Versão | Descrição |
|------------|--------|-----------|
| **Go** | 1.22+ | Linguagem principal do backend |
| **Gorilla Mux** | v1.8.1 | Router HTTP com suporte a parâmetros e sub-routers |
| **pgx / pgxpool** | v5.9.2 | Driver PostgreSQL nativo com pool de conexões |
| **bcrypt** (golang.org/x/crypto) | v0.51.0 | Hash seguro de senhas |
| **godotenv** | v1.5.1 | Carregamento de variáveis de ambiente via arquivo `.env` |
| **log/slog** | stdlib | Logging estruturado em JSON (Go 1.21+) |
| **PostgreSQL** | — | Banco de dados relacional |

## Arquitetura

O projeto segue o padrão **Clean Architecture**, com dependências unidirecionais aplicadas de verdade — cada camada depende apenas de interfaces, nunca de tipos concretos das camadas adjacentes:

```
┌──────────────────────────────────────────────────────┐
│               Middleware Stack                        │  ← RequestID, Logger, Recovery, CORS...
├──────────────────────────────────────────────────────┤
│               Handler (API)                           │  ← Recebe HTTP, parseia DTOs
│               depende de: BankService (interface)     │
├──────────────────────────────────────────────────────┤
│               Use Case (Service)                      │  ← Regras de negócio / orquestração
│               depende de: Repository (interface)      │
├──────────────────────────────────────────────────────┤
│               Domain (Entities + Errors)              │  ← Entidades e erros do domínio
│               sem dependências externas               │
├──────────────────────────────────────────────────────┤
│               Repository (Interface + PostgresRepo)   │  ← Acesso a dados / persistência
│               implementa: Repository interface        │
├──────────────────────────────────────────────────────┤
│               Validation (Pure Functions)             │  ← Validações puras (CPF, email, senha)
└──────────────────────────────────────────────────────┘
```

**Fluxo de dependência:** `main → Handler(BankService) → Service(Repository) → PostgresRepo`

O `main.go` é o único ponto que conhece os tipos concretos — ele monta as camadas e injeta as dependências. Todo o restante da aplicação opera sobre interfaces, o que garante testabilidade e isolamento total entre camadas.

### Decisões de design

- **`Repository` interface unificada** — uma única interface agrupa todas as operações de persistência. O `Service` a recebe por injeção de dependência, sem nunca importar o `PostgresRepo` diretamente.
- **`BankService` interface** — o `Handler` depende desta interface, não do `*Service`. Isso permite mockar o serviço inteiramente em testes.
- **`var _ BankService = (*Service)(nil)`** — verificação de implementação em compile-time. Se o `Service` deixar de implementar o contrato, o build quebra imediatamente.
- **`pgxpool.Pool`** — pool de conexões em vez de conexão única (`pgx.Conn`), essencial para suportar concorrência real em produção.
- **Transações atômicas** — PIX, depósitos e saques executam todas as suas escritas dentro de uma única transação de banco via `ExecTx`, eliminando risco de inconsistência de saldo.

## Middleware

O middleware stack é aplicado globalmente no router, na seguinte ordem:

| Middleware | Função |
|------------|--------|
| `RequestID` | Gera e injeta um ID único (`X-Request-ID`) em cada requisição via context e header de resposta |
| `Logger` | Loga método, path, status HTTP e latência em JSON estruturado via `log/slog` |
| `Recovery` | Captura panics, loga o stack trace completo e devolve `500` sem expor detalhes internos |
| `Timeout` | Define um deadline de 25s por requisição via `context.WithTimeout` |
| `CORS` | Gerencia headers de Cross-Origin de forma centralizada |
| `MaxBodyBytes` | Limita o body a 1 MB para prevenir ataques de payload excessivo |

Exemplo de log gerado por requisição:

```json
{
  "time": "2025-01-15T11:00:00Z",
  "level": "INFO",
  "msg": "request",
  "id": "a3f8c21d9b4e7f01",
  "method": "POST",
  "path": "/api/pix",
  "status": 200,
  "latency_ms": 12,
  "ip": "177.23.45.67:52341"
}
```

## Pré-requisitos

- **Go** 1.22 ou superior instalado
- **PostgreSQL** 12+ em execução
- Banco de dados criado com as tabelas necessárias (veja a seção abaixo)

## Instalação e Execução

### 1. Clone o repositório

```bash
git clone https://github.com/Joshua-likesjs/Delta-Bank-Backend-In-Golang.git
cd Delta-Bank-Backend-In-Golang
```

### 2. Instale as dependências

```bash
go mod tidy
```

### 3. Configure o banco de dados

Crie o banco de dados PostgreSQL e execute os seguintes comandos SQL para criar as tabelas:

```sql
CREATE TABLE contas (
    cpf            VARCHAR(11) PRIMARY KEY,
    owner          VARCHAR(100) NOT NULL,
    balance        INTEGER NOT NULL DEFAULT 0,
    senha          VARCHAR(255) NOT NULL,
    limite_diario  INTEGER NOT NULL DEFAULT 5000000,
    created_at     TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP
);

CREATE TABLE transacoes (
    id                      SERIAL PRIMARY KEY,
    conta_cpf_remetente     VARCHAR(11) NOT NULL REFERENCES contas(cpf),
    conta_cpf_destinatario  VARCHAR(11) NOT NULL REFERENCES contas(cpf),
    tipo                    VARCHAR(20) NOT NULL,
    valor_centavos          INTEGER NOT NULL,
    created_at              TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE chaves_pix (
    id         SERIAL PRIMARY KEY,
    conta_cpf  VARCHAR(11) NOT NULL REFERENCES contas(cpf),
    tipo       VARCHAR(20) NOT NULL,
    valor      VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### 4. Configure as variáveis de ambiente

Crie um arquivo `.env` na raiz do projeto:

```env
DATABASE_URL=postgres://usuario:senha@localhost:5432/nome_do_banco
PORT=8080
```

Ou exporte diretamente no shell:

```bash
export DATABASE_URL="postgres://usuario:senha@localhost:5432/nome_do_banco"
export PORT="8080"
```

### 5. Execute o servidor

```bash
go run cmd/server/main.go
```

O servidor iniciará na porta configurada (padrão: `8080`):

```
🌐 Delta Bank API — porta :8080
```

## Variáveis de Ambiente

| Variável | Obrigatória | Padrão | Descrição |
|----------|-------------|--------|-----------|
| `DATABASE_URL` | Sim | — | String de conexão PostgreSQL (`postgres://user:pass@host:port/db`) |
| `PORT` | Não | `8080` | Porta em que o servidor HTTP escuta |

## Estrutura do Banco de Dados

O sistema utiliza **3 tabelas** principais:

### `contas` — Contas bancárias
| Coluna | Tipo | Descrição |
|--------|------|-----------|
| `cpf` | VARCHAR(11) PK | CPF do titular (sem pontuação) |
| `owner` | VARCHAR(100) | Nome do titular |
| `balance` | INTEGER | Saldo em centavos (R$ 1,00 = 100) |
| `senha` | VARCHAR(255) | Hash bcrypt da senha |
| `limite_diario` | INTEGER | Limite diário de PIX em centavos |
| `created_at` | TIMESTAMP | Data de criação da conta |
| `updated_at` | TIMESTAMP | Data da última atualização |

### `transacoes` — Movimentações financeiras
| Coluna | Tipo | Descrição |
|--------|------|-----------|
| `id` | SERIAL PK | ID auto-incremento |
| `conta_cpf_remetente` | VARCHAR(11) FK | CPF de quem enviou |
| `conta_cpf_destinatario` | VARCHAR(11) FK | CPF de quem recebeu |
| `tipo` | VARCHAR(20) | Tipo: `TRANSACAO_PIX`, `DEPOSITO`, `SAQUE` |
| `valor_centavos` | INTEGER | Valor em centavos |
| `created_at` | TIMESTAMP | Data/hora da transação |

### `chaves_pix` — Chaves PIX cadastradas
| Coluna | Tipo | Descrição |
|--------|------|-----------|
| `id` | SERIAL PK | ID auto-incremento |
| `conta_cpf` | VARCHAR(11) FK | CPF do titular da chave |
| `tipo` | VARCHAR(20) | Tipo: `CPF`, `EMAIL`, `TELEFONE`, `ALEATORIA` |
| `valor` | VARCHAR(100) UNIQUE | Valor da chave PIX |
| `created_at` | TIMESTAMP | Data de cadastro |

## Endpoints da API

Todos os endpoints estão sob o prefixo `/api`.

### Saúde

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/health` | Verifica se a API está no ar (sem prefixo `/api`) |

### Contas

| Método | Rota | Descrição |
|--------|------|-----------|
| `POST` | `/api/contas` | Criar nova conta bancária |
| `POST` | `/api/login` | Autenticar login por CPF e senha |
| `GET` | `/api/saldo/{cpf}` | Consultar saldo da conta |
| `GET` | `/api/extrato/{cpf}` | Obter extrato de transações |
| `PUT` | `/api/dados` | Editar nome e/ou limite diário |
| `PUT` | `/api/senha` | Alterar senha da conta |

### Operações Financeiras

| Método | Rota | Descrição |
|--------|------|-----------|
| `POST` | `/api/pix` | Realizar transferência PIX |
| `POST` | `/api/depositar` | Realizar depósito |
| `POST` | `/api/sacar` | Realizar saque |

### Chaves PIX

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/api/chaves-pix/{cpf}` | Listar chaves PIX da conta |
| `POST` | `/api/chaves-pix` | Adicionar nova chave PIX |
| `DELETE` | `/api/chaves-pix` | Remover chave PIX |

## Exemplos de Requisição

### Criar Conta

```bash
curl -X POST https://delta-bank-backend-in-golang-production.up.railway.app/api/contas \
  -H "Content-Type: application/json" \
  -d '{
    "nome": "João Silva",
    "cpf": "12345678909",
    "senha": "minhasenha",
    "saldo_inicial": 100.00
  }'
```

**Resposta (`201 Created`):**
```json
{
  "sucesso": true,
  "mensagem": "Conta criada!",
  "dados": {
    "cpf": "12345678909",
    "nome": "João Silva",
    "saldo_centavos": 10000,
    "limite_diario": 5000000,
    "criada_em": "2025-01-15T10:30:00Z"
  }
}
```

### Login

```bash
curl -X POST https://delta-bank-backend-in-golang-production.up.railway.app/api/login \
  -H "Content-Type: application/json" \
  -d '{
    "cpf": "12345678909",
    "senha": "minhasenha"
  }'
```

**Resposta (`200 OK`):**
```json
{
  "sucesso": true,
  "mensagem": "Login OK!",
  "dados": {
    "cpf": "12345678909",
    "nome": "João Silva",
    "saldo_centavos": 10000
  }
}
```

### Consultar Saldo

```bash
curl https://delta-bank-backend-in-golang-production.up.railway.app/api/saldo/12345678909
```

**Resposta (`200 OK`):**
```json
{
  "sucesso": true,
  "dados": {
    "Conta": {
      "cpf": "12345678909",
      "nome": "João Silva",
      "saldo_centavos": 10000,
      "limite_diario": 5000000
    },
    "QtdChaves": 2
  }
}
```

### Fazer PIX

```bash
curl -X POST https://delta-bank-backend-in-golang-production.up.railway.app/api/pix \
  -H "Content-Type: application/json" \
  -d '{
    "cpf_origem": "12345678909",
    "cpf_destino": "98765432100",
    "valor": 50.00
  }'
```

**Resposta (`200 OK`):**
```json
{
  "sucesso": true,
  "mensagem": "PIX realizado!",
  "dados": {
    "id": 15,
    "tipo": "TRANSACAO_PIX",
    "valor_centavos": 5000,
    "cpf_remetente": "12345678909",
    "cpf_destinatario": "98765432100",
    "nome_remetente": "João Silva",
    "nome_destinatario": "Maria Souza",
    "data_hora": "2025-01-15T11:00:00Z"
  }
}
```

### Depositar

```bash
curl -X POST https://delta-bank-backend-in-golang-production.up.railway.app/api/depositar \
  -H "Content-Type: application/json" \
  -d '{
    "cpf": "12345678909",
    "valor": 200.00
  }'
```

**Resposta (`200 OK`):**
```json
{
  "sucesso": true,
  "mensagem": "Depósito realizado!",
  "dados": {
    "novo_saldo_centavos": 30000
  }
}
```

### Sacar

```bash
curl -X POST https://delta-bank-backend-in-golang-production.up.railway.app/api/sacar \
  -H "Content-Type: application/json" \
  -d '{
    "cpf": "12345678909",
    "valor": 50.00
  }'
```

**Resposta (`200 OK`):**
```json
{
  "sucesso": true,
  "mensagem": "Saque realizado!",
  "dados": {
    "novo_saldo_centavos": 25000
  }
}
```

### Consultar Extrato

```bash
curl https://delta-bank-backend-in-golang-production.up.railway.app/api/extrato/12345678909
```

**Resposta (`200 OK`):**
```json
{
  "sucesso": true,
  "dados": {
    "transacoes": [
      {
        "id": 15,
        "tipo": "TRANSACAO_PIX",
        "valor_centavos": 5000,
        "cpf_remetente": "12345678909",
        "cpf_destinatario": "98765432100",
        "nome_remetente": "João Silva",
        "nome_destinatario": "Maria Souza",
        "data_hora": "2025-01-15T11:00:00Z"
      }
    ],
    "saldo_atual": 25000
  }
}
```

### Adicionar Chave PIX

```bash
curl -X POST https://delta-bank-backend-in-golang-production.up.railway.app/api/chaves-pix \
  -H "Content-Type: application/json" \
  -d '{
    "cpf": "12345678909",
    "tipo": "EMAIL",
    "valor": "joao@email.com"
  }'
```

**Resposta (`201 Created`):**
```json
{
  "sucesso": true,
  "mensagem": "Chave adicionada!",
  "dados": {
    "id": 3,
    "tipo": "EMAIL",
    "valor": "joao@email.com",
    "cpf_conta": "12345678909",
    "criada_em": "2025-01-15T12:00:00Z"
  }
}
```

### Remover Chave PIX

```bash
curl -X DELETE https://delta-bank-backend-in-golang-production.up.railway.app/api/chaves-pix \
  -H "Content-Type: application/json" \
  -d '{
    "cpf": "12345678909",
    "tipo": "EMAIL",
    "valor": "joao@email.com"
  }'
```

### Alterar Senha

```bash
curl -X PUT https://delta-bank-backend-in-golang-production.up.railway.app/api/senha \
  -H "Content-Type: application/json" \
  -d '{
    "cpf": "12345678909",
    "senha_atual": "minhasenha",
    "nova_senha": "novasenha123"
  }'
```

### Editar Dados da Conta

```bash
curl -X PUT https://delta-bank-backend-in-golang-production.up.railway.app/api/dados \
  -H "Content-Type: application/json" \
  -d '{
    "cpf": "12345678909",
    "novo_nome": "João Silva Santos",
    "novo_limite_diario": 3000.00
  }'
```

> Os campos `novo_nome` e `novo_limite_diario` são opcionais — envie apenas o que deseja alterar. Os dados não informados são preservados automaticamente.

## Validações e Regras de Negócio

### Contas
- **Nome**: mínimo de 3 caracteres (espaços ignorados)
- **CPF**: validação completa com dígitos verificadores (algoritmo oficial da Receita Federal)
- **Senha**: mínimo de 6 caracteres; armazenada como hash bcrypt (nunca em texto plano)
- **Saldo inicial**: mínimo de R$ 10,00 para abertura de conta
- **CPF duplicado**: não é permitido cadastrar o mesmo CPF mais de uma vez

### PIX
- **Valor mínimo**: R$ 1,00 por transferência
- **Limite diário**: cada conta possui um limite diário de PIX (padrão: R$ 50.000,00)
- **Auto-transferência**: não é permitido enviar PIX para a própria conta
- **Saldo insuficiente**: a transferência é bloqueada se o saldo não cobrir o valor
- **Atomicidade**: débito, crédito e registro da transação ocorrem em uma única transação de banco — nenhuma operação parcial é possível

### Depósitos
- Valor deve estar entre R$ 0,01 e R$ 10.000,00
- Operação atômica: saldo e registro atualizados na mesma transação

### Saques
- Valor deve estar entre R$ 0,01 e R$ 2.000,00
- O valor deve ser múltiplo de R$ 10,00 (notas de caixa eletrônico)
- Saldo deve ser suficiente para cobrir o saque
- Operação atômica: saldo e registro atualizados na mesma transação

### Chaves PIX
- Tipos suportados: `CPF`, `EMAIL`, `TELEFONE`, `ALEATORIA`
- Validação de email com regex
- Validação de telefone brasileiro com verificação de DDD válido
- Chave aleatória gerada como UUID v4 criptograficamente seguro (`crypto/rand`)
- Limite de 5 chaves por conta
- Cada valor de chave é único no sistema

## Tratamento de Erros

Todos os erros seguem um formato padrão de resposta com o status HTTP correto:

```json
{
  "sucesso": false,
  "mensagem": "descrição do erro"
}
```

Erros internos nunca expõem detalhes de infraestrutura ao cliente — o handler mapeia qualquer erro não-domínio para `500` com mensagem genérica.

### Códigos de Erro do Domínio

| Mensagem | HTTP | Descrição |
|----------|------|-----------|
| `conta não encontrada` | 404 | CPF não existe no sistema |
| `CPF já cadastrado` | 409 | CPF já possui conta |
| `CPF inválido` | 400 | CPF falhou na validação dos dígitos |
| `nome deve ter no mínimo 3 caracteres` | 400 | Nome muito curto |
| `senha deve ter no mínimo 6 caracteres` | 400 | Senha muito curta |
| `nova senha deve ser diferente da atual` | 400 | Tentativa de reusar a mesma senha |
| `senha incorreta` | 401 | Senha não confere com o hash |
| `conta bloqueada por muitas tentativas` | 423 | Conta temporariamente bloqueada |
| `saldo insuficiente` | 400 | Saldo menor que o valor da operação |
| `limite diário excedido` | 400 | Limite diário de PIX atingido |
| `valor inválido` | 400 | Valor fora do intervalo permitido |
| `destinatário não encontrado` | 404 | CPF do destinatário não existe |
| `não pode transferir para si mesmo` | 400 | PIX para o próprio CPF |
| `chave PIX já cadastrada no sistema` | 409 | Valor da chave já existe |
| `limite de 5 chaves PIX atingido` | 400 | Conta já tem 5 chaves |
| `chave PIX não encontrada` | 404 | Chave não existe para a conta |
| `corpo da requisição inválido` | 400 | Body JSON malformado ou ausente |
| `erro interno do servidor` | 500 | Erro inesperado no servidor |

## Estrutura de Diretórios

```
Delta-Bank-Backend-In-Golang/
├── cmd/
│   └── server/
│       └── main.go                 # Ponto de entrada: wiring das camadas, pool, middleware, rotas
├── internal/
│   ├── domain/
│   │   ├── entities.go             # Entidades: Conta, Transacao, ChavePix
│   │   └── errors.go               # Erros de domínio tipados com código HTTP
│   ├── middleware/
│   │   └── middleware.go           # RequestID, Logger, Recovery, Timeout, CORS, MaxBodyBytes
│   ├── handler/
│   │   ├── api_handler.go          # Handlers HTTP — depende de BankService (interface)
│   │   └── dto.go                  # DTOs de entrada/saída da API
│   ├── usecase/
│   │   ├── service.go              # Lógica de negócio — depende de Repository (interface)
│   │   └── ports.go                # BankService interface + structs de input/output
│   ├── repository/
│   │   ├── repository.go           # Repository interface unificada
│   │   └── postgres.go             # Implementação PostgreSQL com pgxpool
│   └── validation/
│       └── validators.go           # Funções puras: CPF, email, telefone, bcrypt, UUID
├── go.mod                          # Módulo e dependências Go
├── go.sum                          # Checksums das dependências
└── README.md                       # Este arquivo
```

### Descrição dos Pacotes

- **`cmd/server`** — Composition root da aplicação. Inicializa o `pgxpool`, instancia as camadas na ordem correta (`PostgresRepo → Service → APIHandler`), configura o middleware stack e o router, e gerencia o graceful shutdown com timeout de 15s.

- **`internal/domain`** — O coração da aplicação. Define as entidades de negócio (`Conta`, `Transacao`, `ChavePix`) e os erros de domínio tipados com código HTTP. Esta camada não possui nenhuma dependência externa.

- **`internal/middleware`** — Middleware stack de produção: `RequestID` (rastreabilidade), `Logger` (JSON estruturado via `slog`), `Recovery` (captura panics), `Timeout` (deadline por request), `CORS` (centralizado) e `MaxBodyBytes` (proteção contra payloads excessivos).

- **`internal/handler`** — Camada de apresentação. Os handlers dependem da interface `BankService`, decodificam DTOs, mapeiam erros de domínio para status HTTP corretos e formatam respostas padronizadas (`RespostaAPI`).

- **`internal/usecase`** — Camada de aplicação. Define a interface `BankService` e implementa toda a orquestração das regras de negócio. Depende apenas da interface `Repository`, sem conhecer o PostgreSQL diretamente.

- **`internal/repository`** — Camada de dados. Define a interface `Repository` com todas as operações de persistência e a implementa em `PostgresRepo` via `pgxpool`. Suporta execução transacional atômica via `ExecTx`.

- **`internal/validation`** — Funções puras de validação sem efeitos colaterais: CPF (dígitos verificadores da Receita Federal), email, telefone brasileiro (DDD válido), senha, hash bcrypt e geração de UUID v4 criptograficamente seguro.

---

Feito com Go e Clean Architecture.