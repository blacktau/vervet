<script lang="ts" setup>
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getUriHost, parseUri } from '@/features/server-pane/connectionStrings.ts'

const i18n = useI18n()

const uri = ref('')
const name = ref('')
const connecting = ref(false)
const lastError = ref<{ code: string; detail: string } | null>(null)

const nameTouched = ref(false)

watch(uri, (next) => {
  if (nameTouched.value) {
    return
  }
  const host = getUriHost(next)
  if (host) {
    name.value = host
  }
})

const onNameInput = (value: string) => {
  nameTouched.value = true
  name.value = value
}

const uriValid = computed(() => {
  if (!uri.value) {
    return false
  }
  return parseUri(uri.value).success
})

const canConnect = computed(() => uriValid.value && !connecting.value)

const onConnect = async () => {
  // wired in Task 7
}
</script>

<template>
  <div class="onboarding-wrapper flex-box-v">
    <div class="onboarding-card">
      <h1 class="title">{{ i18n.t('onboarding.welcomeTitle') }}</h1>
      <p class="subtitle">{{ i18n.t('onboarding.welcomeSubtitle') }}</p>

      <n-form label-placement="top">
        <n-form-item :label="i18n.t('onboarding.uriLabel')">
          <n-input
            v-model:value="uri"
            :placeholder="i18n.t('onboarding.uriPlaceholder')"
            data-test="uri-input"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 4 }"
          />
        </n-form-item>

        <n-form-item :label="i18n.t('onboarding.nameLabel')">
          <n-input
            :value="name"
            :placeholder="i18n.t('onboarding.namePlaceholder')"
            data-test="name-input"
            @update:value="onNameInput"
          />
        </n-form-item>

        <n-alert
          v-if="lastError"
          type="error"
          :closable="false"
          :title="i18n.t('onboarding.errorTitle')"
          data-test="error-alert"
        >
          <div>{{ i18n.t('errors.' + lastError.code) }}</div>
          <div v-if="lastError.detail" class="error-detail">{{ lastError.detail }}</div>
        </n-alert>

        <n-button
          :disabled="!canConnect"
          :loading="connecting"
          block
          size="large"
          type="primary"
          data-test="connect-btn"
          @click="onConnect"
        >
          {{ i18n.t('onboarding.connect') }}
        </n-button>

        <div class="advanced-link">
          <n-button text data-test="advanced-link">
            {{ i18n.t('onboarding.advanced') }}
          </n-button>
        </div>
      </n-form>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.onboarding-wrapper {
  height: 100%;
  justify-content: center;
  align-items: center;
  padding: 20px;
}

.onboarding-card {
  width: 100%;
  max-width: 480px;
}

.title {
  font-size: 24px;
  font-weight: 500;
  margin: 0 0 4px;
}

.subtitle {
  margin: 0 0 24px;
  opacity: 0.75;
}

.advanced-link {
  text-align: center;
  margin-top: 12px;
}

.error-detail {
  margin-top: 4px;
  font-size: 12px;
  opacity: 0.8;
}
</style>
