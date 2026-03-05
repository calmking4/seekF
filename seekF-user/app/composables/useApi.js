export const useApi = (url, options = {}) => {
    options.method = options.method || 'POST';
    const state = useAuthState();
    return useFetch(url, {
        baseURL:useRuntimeConfig().public.apiBase,
        ...options,
        onRequest:({request,options})=>{
            const token = state.getToken();
            if(token){
                options.headers.set('Authorization', `Bearer ${token}`)
            }
        },
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
    const state = useAuthState();
    return $fetch(url, {
        baseURL:useRuntimeConfig().public.apiBase,
        ...options,
        onRequest:({request,options})=>{
            const token = state.getToken();
            if(token){
                options.headers.set('Authorization', `Bearer ${token}`)
            }
        },
        onRequestError: (error) => {
            console.error('onRequestError: ', error);
        },
        onResponseError: (error) => {
            console.error('onResponseError: ', error);
        },
    })
}