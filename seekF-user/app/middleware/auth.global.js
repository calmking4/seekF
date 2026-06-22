export default defineNuxtRouteMiddleware((to) => {
    const { getUser } = useAuthState()
    const user = getUser()

    // 首页：根据登录状态重定向
    if (to.path === '/') {
        if (user) {
            return navigateTo('/discover')
        } else {
            return navigateTo('/login')
        }
    }

    // 需要登录的页面列表
    const protectedRoutes = ['/chat', '/aichat', '/contact', '/discover', '/knowledge', '/my']

    // 未登录用户访问受保护页面时，跳转到登录页
    if (!user && protectedRoutes.some(route => to.path.startsWith(route))) {
        return navigateTo('/login')
    }
})
