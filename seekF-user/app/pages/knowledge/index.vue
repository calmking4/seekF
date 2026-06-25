<template>
    <div class="h-full bg-gray-100 p-4">
        <div class="bg-white rounded-lg shadow p-4 h-full flex flex-col">
            <div class="flex items-center justify-between mb-4">
                <h1 class="text-xl font-medium">知识库管理</h1>
                <el-button type="primary" @click="showUploadDialog">
                    <Icon name="uil:plus" class="mr-1" />
                    上传知识文件
                </el-button>
            </div>

            <div v-if="docList.length === 0" class="flex-1 flex items-center justify-center text-gray-400">
                <div class="text-center">
                    <Icon name="uil:folder" class="text-6xl mb-4" />
                    <p>暂无知识文档</p>
                    <p class="text-sm mt-2">上传 .txt 或 .md 文件构建知识库</p>
                </div>
            </div>

            <div v-else class="flex-1 overflow-y-auto">
                <el-table :data="docList" stripe>
                    <el-table-column prop="file_name" label="文件名" min-width="200" />
                    <el-table-column prop="file_type" label="类型" width="80" />
                    <el-table-column prop="chunk_count" label="向量块" width="80" />
                    <el-table-column prop="created_at" label="上传时间" width="180" />
                    <el-table-column label="操作" width="140">
                        <template #default="{ row }">
                            <el-button type="primary" size="small" text @click="handlePreview(row)">
                                预览
                            </el-button>
                            <el-button type="danger" size="small" text @click="handleDelete(row)">
                                删除
                            </el-button>
                        </template>
                    </el-table-column>
                </el-table>
            </div>
        </div>

        <el-dialog v-model="uploadDialogVisible" title="上传知识文件" width="420px">
            <div class="upload-area" @click="triggerUpload">
                <input
                    ref="fileInput"
                    type="file"
                    class="hidden"
                    accept=".txt,.md"
                    @change="handleFileChange"
                />
                <div v-if="!selectedFile" class="upload-placeholder">
                    <Icon name="uil:upload-alt" class="text-4xl text-gray-400 mb-2" />
                    <p class="text-gray-600">点击或拖拽文件到此处上传</p>
                    <p class="text-sm text-gray-400 mt-1">支持 .txt 和 .md 格式</p>
                </div>
                <div v-else class="file-info">
                    <Icon name="uil:file" class="text-4xl text-blue-500" />
                    <div class="file-name">{{ selectedFile.name }}</div>
                    <div class="file-size">{{ formatFileSize(selectedFile.size) }}</div>
                    <div class="file-type">.{{ selectedFile.name.split('.').pop() }}</div>
                </div>
            </div>

            <template #footer>
                <el-button @click="uploadDialogVisible = false">取消</el-button>
                <el-button type="primary" :loading="uploading" :disabled="!selectedFile" @click="handleUpload">
                    上传
                </el-button>
            </template>
        </el-dialog>

        <el-dialog v-model="previewDialogVisible" title="文档预览" width="700px">
            <div class="preview-content">
                <pre>{{ previewContent }}</pre>
            </div>
            <template #footer>
                <el-button @click="previewDialogVisible = false">关闭</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup>
// 页面级 SEO
useSeoMeta({
  title: '知识库管理',
  description: '上传和管理您的知识文档，构建专属 AI 知识库。',
})

const knowledge = useKnowledge()

const docList = ref([])
const uploadDialogVisible = ref(false)
const selectedFile = ref(null)
const uploading = ref(false)
const uploadRef = ref()
const fileInput = ref()
const previewDialogVisible = ref(false)
const previewContent = ref('')

const loadDocList = async () => {
    const list = await knowledge.getDocumentList()
    docList.value = list.map(item => ({
        file_name: item.file_name,
        file_url: item.file_url,
        file_type: item.file_type,
        chunk_count: item.chunk_count,
        created_at: item.created_at,
        uuid: item.uuid
    }))
}

const showUploadDialog = () => {
    selectedFile.value = null
    if (fileInput.value) {
        fileInput.value.value = ''
    }
    uploadDialogVisible.value = true
}

const triggerUpload = () => {
    fileInput.value?.click()
}

const handleFileChange = (event) => {
    const file = event.target.files?.[0]
    if (file) {
        selectedFile.value = file
    }
}

const formatFileSize = (bytes) => {
    if (bytes < 1024) return bytes + ' B'
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

const handlePreview = async (row) => {
    const content = await knowledge.getDocumentContent(row.uuid)
    if (content) {
        previewContent.value = content
        previewDialogVisible.value = true
    } else {
        ElMessage.error('获取文件内容失败')
    }
}

const handleUpload = async () => {
    if (!selectedFile.value) {
        ElMessage.warning('请选择文件')
        return
    }

    const file = selectedFile.value
    const fileName = file.name
    const fileType = fileName.split('.').pop().toLowerCase()

    if (!['txt', 'md'].includes(fileType)) {
        ElMessage.warning('仅支持 .txt 和 .md 文件')
        return
    }

    uploading.value = true

    try {
        const formData = new FormData()
        formData.append('file', file)
        formData.append('fileType', 'knowledge_doc')

        const config = useRuntimeConfig()
        const apiBase = config.public.apiBase ? config.public.apiBase.replace(/\/$/, '') : 'http://localhost:8080'

        const res = await fetch(`${apiBase}/user/file/upload`, {
            method: 'POST',
            body: formData,
            credentials: 'include'
        })

        const data = await res.json()

        if (data.code === 200) {
            const fileUrl = data.data.url

            const result = await knowledge.addDocument(fileName, fileUrl, fileType)
            if (result) {
                ElMessage.success('上传成功')
                uploadDialogVisible.value = false
                await loadDocList()
            } else {
                ElMessage.error('添加知识文档失败')
            }
        } else {
            ElMessage.error('文件上传失败')
        }
    } catch (error) {
        console.error('上传失败:', error)
        ElMessage.error('上传失败')
    } finally {
        uploading.value = false
    }
}

const handleDelete = async (row) => {
    try {
        await ElMessageBox.confirm('确定要删除这个知识文档吗？', '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
        })

        const success = await knowledge.removeDocument(row.uuid)
        if (success) {
            ElMessage.success('删除成功')
            await loadDocList()
        } else {
            ElMessage.error('删除失败')
        }
    } catch {
        // 用户取消
    }
}

onMounted(async () => {
    await loadDocList()
})
</script>

<style scoped>
.upload-area {
    border: 2px dashed #dcdfe6;
    border-radius: 8px;
    padding: 40px 20px;
    text-align: center;
    cursor: pointer;
    transition: all 0.3s;
}
.upload-area:hover {
    border-color: #409eff;
    background-color: #f5f7fa;
}
.upload-placeholder {
    pointer-events: none;
}
.file-info {
    display: flex;
    align-items: center;
    gap: 12px;
}
.file-info .file-name {
    flex: 1;
    text-align: left;
    font-weight: 500;
    color: #303133;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}
.file-info .file-size {
    color: #909399;
    font-size: 13px;
}
.file-info .file-type {
    background: #ecf5ff;
    color: #409eff;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 500;
}

.preview-content {
    max-height: 500px;
    overflow-y: auto;
    background: #f8f9fa;
    padding: 16px;
    border-radius: 6px;
}
.preview-content pre {
    white-space: pre-wrap;
    word-wrap: break-word;
    font-family: 'Microsoft YaHei', sans-serif;
    font-size: 14px;
    line-height: 1.6;
    margin: 0;
}
</style>