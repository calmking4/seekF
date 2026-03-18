export const useAuthState = () => {
    // 创建一个响应式状态并设置默认值
    const user = useState('user', () => null)
    const token = useState('token', () => null)

    // 从 cookie 中读取 token
    const tokencookie = useCookie('token')
    token.value = tokencookie.value || null

    // 仅更新状态，不设置 cookie（由后端处理）
    const setUser = (value) => {
        user.value = value
    }
    
    const setToken = (value) => {
        token.value = value
    }
    
    const getUser = () => {
        return user.value
    }
    
    const getToken = () => {
        return token.value
    }

    // 清除状态
    const clear = ()=>{
        setUser(null)
        setToken(null)
    }

    
    return {
        setUser,getUser,
        setToken,getToken,
        clear,
    }
}