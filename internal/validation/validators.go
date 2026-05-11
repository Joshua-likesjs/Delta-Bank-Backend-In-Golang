package validation

import (
    "fmt"
    "regexp"
    "strings"
    "time"

    "golang.org/x/crypto/bcrypt"
)

// ============================================================
//  VALIDAÇÕES (Pure Functions - Sem dependências!)
// ============================================================

// ValidadorCPF valida e formata CPF
func ValidadorCPF(cpf string) (string, bool) {
    c := removerNaoNumericos(cpf)
    if len(c) != 11 || todosDigitosIguais(c) {
        return "", false
    }
    d1 := calcDigitoVerificador(c[:9], 10)
    if d1 != int(c[9]-'0') { return "", false }
    d2 := calcDigitoVerificador(c[:10], 11)
    if d2 != int(c[10]-'0') { return "", false }
    return c, true
}

// ValidadorEmail valida formato de email
func ValidadorEmail(email string) (string, bool) {
    e := strings.TrimSpace(strings.ToLower(email))
    if !regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`).MatchString(e) {
        return "", false
    }
    parts := strings.Split(e, "@")
    if len(parts) != 2 || len(parts[1]) < 5 || !strings.Contains(parts[1], ".") {
        return "", false
    }
    return e, true
}

// ValidadorTelefone valida telefone brasileiro
func ValidadorTelefone(tel string) (string, bool) {
    n := removerNaoNumericos(tel)
    if len(n) > 11 && n[:2] == "55" { n = n[2:] }
    if len(n) != 10 && len(n) != 11 { return "", false }
    if !validarDDD(n[:2]) || n[2] == '0' || (len(n) == 11 && n[2] != '9') {
        return "", false
    }
    return n, true
}

// ValidadorSenha verifica requisitos mínimos
func ValidadorSenha(senha string) bool {
    return len(senha) >= 6
}

// HashSenha gera hash bcrypt seguro
func HashSenha(senha string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
    if err != nil {
        return "", fmt.Errorf("erro ao hash senha: %w", err)
    }
    return string(bytes), nil
}

// VerificarSenha compara senha com hash
func VerificarSenha(senha, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(senha))
    return err == nil
}

// Formatações
func FormatarCPF(cpf string) string {
    if len(cpf) == 11 { return cpf[:3] + "." + cpf[3:6] + "." + cpf[6:9] + "-" + cpf[9:] }
    return cpf
}

func FormatarTelefone(tel string) string {
    if len(tel) == 11 { return "(" + tel[:2] + ") " + tel[2:7] + "-" + tel[7:] }
    if len(tel) == 10 { return "(" + tel[:2] + ") " + tel[2:6] + "-" + tel[6:] }
    return tel
}

// GerarChaveAleatoria gera UUID v4 para chave PIX
func GerarChaveAleatoria() string {
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

// Funções auxiliares privadas
func removerNaoNumericos(s string) string {
    return regexp.MustCompile(`[^0-9]`).ReplaceAllString(s, "")
}

func todosDigitosIguais(s string) bool {
    for i := 1; i < len(s); i++ {
        if s[i] != s[0] { return false }
    }
    return true
}

func calcDigitoVerificador(d string, peso int) int {
    soma := 0
    w := peso
    for i := 0; i < len(d); i++ {
        soma += int(d[i]-'0') * w
        w--
    }
    r := soma % 11
    if r < 2 { return 0 }
    return 11 - r
}

func validarDDD(ddd string) bool {
    ddds := map[string]bool{
        "11":true,"12":true,"13":true,"14":true,"15":true,"16":true,"17":true,"18":true,"19":true,
        "21":true,"22":true,"24":true,"27":true,"28":true,"31":true,"32":true,"33":true,"34":true,
        "35":true,"37":true,"38":true,"41":true,"42":true,"43":true,"44":true,"45":true,"46":true,
        "47":true,"48":true,"49":true,"51":true,"53":true,"54":true,"55":true,"61":true,"62":true,
        "63":true,"64":true,"65":true,"66":true,"67":true,"71":true,"73":true,"74":true,"75":true,
        "77":true,"78":true,"79":true,"81":true,"82":true,"83":true,"84":true,"85":true,"86":true,
        "87":true,"88":true,"89":true,"91":true,"92":true,"93":true,"94":true,"95":true,"96":true,
        "97":true,"98":true,"99":true,
    }
    return ddds[ddd]
}