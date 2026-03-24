export const useAuthState = () => {
    // 创建一个响应式状态并设置默认值
    const user = useState('user', () => null)

    const usercookie = useCookie('user')
    user.value = usercookie.value

    const maxAge = 60 * 60 * 24

    const setUser = (value) => {
        user.value = value
        usercookie.value = value
        usercookie.maxAge = maxAge
    }
    
    const getUser = () => {
        return user.value
    }

    // 清除状态
    const clear = ()=>{
        setUser(null)
    }

    
    return {
        setUser,getUser,
        clear,
    }
}