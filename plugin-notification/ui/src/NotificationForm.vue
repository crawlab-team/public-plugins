<template>
  <cl-form :model="internalForm" ref="formRef">
    <cl-form-item :span="2" :label="t('form.name')" prop="name" required>
      <el-input v-model="internalForm.name" :placeholder="t('form.name')" @change="onChange"/>
    </cl-form-item>
    <cl-form-item :span="2" :label="t('form.description')" prop="description">
      <el-input
          v-model="internalForm.description"
          type="textarea"
          :placeholder="t('form.description')"
          @change="onChange"
      />
    </cl-form-item>
    <cl-form-item :span="2" :label="t('form.type')" prop="type">
      <el-select v-model="internalForm.type" @change="onChange">
        <el-option value="mail" :label="t('notifications.type.mail')"/>
        <el-option value="mobile" :label="t('notifications.type.mobile')"/>
      </el-select>
    </cl-form-item>
    <cl-form-item :span="2" :label="t('form.enabled')" prop="enabled">
      <cl-switch v-model="internalForm.enabled" @change="onChange"/>
    </cl-form-item>

    <template v-if="internalForm.type === 'mail'">
      <cl-form-item :span="2" :label="t('form.mail.smtp.server')" prop="mail.server" required>
        <el-input
            v-model="internalForm.mail.server"
            :placeholder="t('form.mail.smtp.server')"
            @change="onChange"
        />
      </cl-form-item>
      <cl-form-item :span="2" :label="t('form.mail.smtp.port')" prop="mail.port" required>
        <el-input
            v-model="internalForm.mail.port"
            :placeholder="t('form.mail.smtp.port')"
            @change="onChange"
        />
      </cl-form-item>
      <cl-form-item :span="2" :label="t('form.mail.smtp.user')" prop="mail.user">
        <el-input
            v-model="internalForm.mail.user"
            :placeholder="t('form.mail.smtp.user')"
            @change="onChange"
        />
      </cl-form-item>
      <cl-form-item :span="2" :label="t('form.mail.smtp.password')" prop="mail.password">
        <el-input
            v-model="internalForm.mail.password"
            :placeholder="t('form.mail.smtp.password')"
            @change="onChange"
        />
      </cl-form-item>
      <cl-form-item :span="2" :label="t('form.mail.smtp.sender.email')" prop="mail.sender_email">
        <el-input
            v-model="internalForm.mail.sender_email"
            :placeholder="t('form.mail.smtp.sender.email')"
            @change="onChange"
        />
      </cl-form-item>
      <cl-form-item :span="2" :label="t('form.mail.smtp.sender.identity')" prop="mail.sender_identity">
        <el-input
            v-model="internalForm.mail.sender_identity"
            :placeholder="t('form.mail.smtp.sender.identity')"
            @change="onChange"
        />
      </cl-form-item>
      <cl-form-item :span="2" :label="t('form.mail.to')" prop="mail.to" required>
        <el-input
            v-model="internalForm.mail.to"
            :placeholder="t('form.mail.to')"
            @change="onChange"
        />
      </cl-form-item>
      <cl-form-item :span="2" :label="t('form.mail.cc')" prop="mail.cc">
        <el-input
            v-model="internalForm.mail.cc"
            :placeholder="t('form.mail.cc')"
            @change="onChange"
        />
      </cl-form-item>
    </template>

    <template v-else-if="internalForm.type === 'mobile'">
      <cl-form-item :span="4" :label="t('form.mobile.webhook')" prop="mobile.webhook">
        <el-input
            v-model="internalForm.mobile.webhook"
            :placeholder="t('form.mobile.webhook')"
            @change="onChange"
        />
      </cl-form-item>
    </template>

  </cl-form>
</template>

<script lang="ts">
import {defineComponent, onMounted, ref, watch} from 'vue';

const pluginName = 'notification';
const t = (path) => window['_tp'](pluginName, path);
const _t = window['_t'];

export default defineComponent({
  name: 'NotificationForm',
  props: {
    modelValue: {
      type: Object,
      default: () => {
        return {};
      }
    },
  },
  emits: [
    'update:modelValue',
  ],
  setup(props, {emit}) {
    const formRef = ref();

    const internalForm = ref({
      name: '',
      description: '',
      type: 'mail',
      enabled: true,
      global: true,
      mail: {
        server: '',
        port: '465',
        user: '',
        password: '',
        sender_email: '',
        sender_identity: '',
        title: '',
        template: '',
        to: '',
        cc: '',
      },
      mobile: {
        webhook: '',
        title: '',
        template: '',
      },
    });

    onMounted(() => {
      // eslint-disable-next-line @typescript-eslint/ban-ts-ignore
      // @ts-ignore
      internalForm.value = props.modelValue;
    });

    watch(() => props.modelValue, () => {
      // eslint-disable-next-line @typescript-eslint/ban-ts-ignore
      // @ts-ignore
      internalForm.value = props.modelValue;
    });

    const onChange = () => {
      emit('update:modeValue', internalForm.value);
    };

    const validate = async () => {
      await formRef.value.validate();
    };

    return {
      formRef,
      internalForm,
      onChange,
      validate,
      t,
    };
  },
});
</script>

<style scoped>

</style>
