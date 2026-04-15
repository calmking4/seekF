export const useKnowledge = () => {
    const config = useRuntimeConfig()

    const addDocument = async (fileName, fileUrl, fileType) => {
        try {
            const res = await useApi$('/user/knowledge/add', {
                method: 'POST',
                body: {
                    file_name: fileName,
                    file_url: fileUrl,
                    file_type: fileType
                }
            })
            if (res?.code === 200) {
                return res.data
            }
            return null
        } catch (error) {
            console.error('添加知识文档失败:', error)
            return null
        }
    }

    const getDocumentList = async () => {
        try {
            const res = await useApi$('/user/knowledge/list', {
                method: 'POST'
            })
            if (res?.code === 200) {
                return res.data.list || []
            }
            return []
        } catch (error) {
            console.error('获取知识文档列表失败:', error)
            return []
        }
    }

    const removeDocument = async (uuid) => {
        try {
            const res = await useApi$('/user/knowledge/remove', {
                method: 'POST',
                body: { uuid }
            })
            if (res?.code === 200) {
                return true
            }
            return false
        } catch (error) {
            console.error('删除知识文档失败:', error)
            return false
        }
    }

    const getDocumentContent = async (uuid) => {
        try {
            const res = await useApi$('/user/knowledge/content', {
                method: 'POST',
                body: { uuid }
            })
            if (res?.code === 200) {
                return res.data.content
            }
            return null
        } catch (error) {
            console.error('获取文件内容失败:', error)
            return null
        }
    }

    return {
        addDocument,
        getDocumentList,
        removeDocument,
        getDocumentContent
    }
}