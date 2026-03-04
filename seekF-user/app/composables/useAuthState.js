export const useAuthState = () => {
    // 创建一个响应式状态并设置默认值
    const user = useState('user', () => null)
    const token = useState('token', () => null)

    const usercookie = useCookie('user')
    user.value = usercookie.value || null
    const tokencookie = useCookie('token')
    token.value = tokencookie.value || null

    const maxAge = 60 * 60 // 一个小时
    const setUser = (value) => {
        user.value = value

        const usercookie = useCookie('user',{maxAge: maxAge})
        usercookie.value = value
    }
    
    const setToken = (value) => {
        token.value = value

        const tokencookie = useCookie('token',{maxAge: maxAge})
        tokencookie.value = value
    }
    const getUser = () => {
        return user.value
    }
    const getToken = () => {
        return token.value
    }

    const clearUser = ()=>{
        setUser(null)
        setToken(null)
    }
    return {
        // user,
        setUser,getUser,
        setToken,getToken,
        clearUser
    }
}