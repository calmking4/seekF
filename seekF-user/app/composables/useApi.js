export const useApi = (url, options = {}) => {
    options.method = options.method || 'POST';
    return useFetch(url, {
        baseURL:useRuntimeConfig().public.apiBase,
        ...options,
        credentials: "include",
        // onRequest:({request,options})=>{
        //     const token = state.getToken();
        //     if(token){
        //         options.headers.set('Authorization', `Bearer ${token}`)
        //     }
        // },
        onRequestError: (error) => {
            console.error('onRequestError: ', error);
        },
        onResponseError: (error) => {
            console.error('onResponseError: ', error);
        },
    })
}

export const useApi$ = (url, options = {}) => {
    options.method = options.method || 'POST';
    return $fetch(url, {
        baseURL:useRuntimeConfig().public.apiBase,
        ...options,
        credentials: "include",
        onRequestError: (error) => {
            console.error('onRequestError: ', error);
        },
        onResponseError: (error) => {
            console.error('onResponseError: ', error);
            if (error.response && error.response._data && error.response._data.code === 401) {
                ElMessage.error('请先登录');
            }
        },
    })
}