export const useApi = (url, options = {}) => {
    options.method = options.method || 'POST';
    return useFetch(url, {
        baseURL: useRuntimeConfig().public.apiBase,
        ...options,
        credentials: "include",
        onRequestError: (error) => {
            console.error('onRequestError: ', error);
            ElMessage.error('网络请求失败，请检查网络连接');
        },
        onResponseError: (error) => {
            console.error('onResponseError: ', error);
            handleApiError(error);
        },
    })
}

export const useApi$ = (url, options = {}) => {
    options.method = options.method || 'POST';
    return $fetch(url, {
        baseURL: useRuntimeConfig().public.apiBase,
        ...options,
        credentials: "include",
        onRequestError: (error) => {
            console.error('onRequestError: ', error);
            ElMessage.error('网络请求失败，请检查网络连接');
        },
        onResponseError: (error) => {
            console.error('onResponseError: ', error);
            handleApiError(error);
        },
    })
}

// 统一错误处理函数
const handleApiError = (error) => {
    const status = error.response?.status
    const message = error.response?._data?.message || '请求失败'

    switch (status) {
        case 401:
            ElMessage.error('请先登录')
            navigateTo('/login')
            break
        case 403:
            ElMessage.error('无权限访问')
            break
        case 404:
            ElMessage.error('请求的资源不存在')
            break
        case 500:
            ElMessage.error('服务器错误，请稍后重试')
            break
        default:
            ElMessage.error(message)
    }
}