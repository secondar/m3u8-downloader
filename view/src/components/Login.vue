<template>
  <div class="login-container">
    <div class="title">M3u8Download</div>
    <div class="form">
      <div>Token</div>
      <div class="input"><el-input v-model="token" style="width: 240px" placeholder="Please input Token" /></div>
      <div><el-button type="primary" @click="handleLogin" plain>登入</el-button></div>
    </div>
    <div class="tip"><el-text type="info">如果你忘记了Token，请自行使用Sqlite修改data/db.db下的conf表key为Token的value值</el-text></div>
  </div>
</template>
<script setup lang="ts">
import { checkToken } from "../api/api";
import type { Params } from "../api/api";
import { ElMessageBox, ElLoading } from 'element-plus'
import { ref } from 'vue'
import { setToken } from '../utils/auth'
const token = ref('')
const emit = defineEmits(['ret'])
const handleLogin = () => {
  const loading = ElLoading.service({
    lock: true,
    text: 'Loading...',
    background: 'rgba(0, 0, 0, 0.7)',
  })
  var params: Params = {
    "token": token.value
  }
  checkToken(params).then(res => {
    if (res.code == 200) {
      setToken(token.value, true)
      emit('ret')
    } else {
      ElMessageBox.alert(res.msg, '发生错误', {
        confirmButtonText: '确认'
      })
    }
  }).catch(err => {
    ElMessageBox.alert(err.response != undefined && err.response.data != undefined ? err.response.data : err.message, '发生错误', {
      confirmButtonText: '确认'
    })
  }).finally(() => {
    loading.close()
  })
}
</script>
<style scoped></style>
