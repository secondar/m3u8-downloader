<template>
  <Login v-if="!Token || Token === ''" @ret="loginRet" />
  <Control v-else></Control>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import Login from './components/Login.vue'
import Control from './components/Control.vue'
import { getToken } from './utils/auth'
const Token = ref(getToken())
const loginRet = () => {
  Token.value = getToken()
}
</script>