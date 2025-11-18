import Cookies from 'js-cookie'
const TokenKey = 'Token'
export function setToken(Token: string, RememberMe: boolean) {
    sessionStorage.setItem(TokenKey, Token)
    if (RememberMe) {
        Cookies.set(TokenKey, Token)
    }
}
export function getToken() {
    var Token: string | null | undefined = sessionStorage.getItem(TokenKey)
    if (Token == null) {
        Token = Cookies.get(TokenKey)
    }
    return Token
}
export function clearToken() {
    sessionStorage.clear()
    Cookies.remove(TokenKey)
}